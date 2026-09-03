// Package setup is the whole flow: detect harnesses, plan what the skills, the
// letter, and the tools need, ask the human, apply, and report.
package setup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/janiorvalle/jstack/internal/harness"
	"github.com/janiorvalle/jstack/internal/letter"
	"github.com/janiorvalle/jstack/internal/prompt"
	"github.com/janiorvalle/jstack/internal/skills"
	"github.com/janiorvalle/jstack/internal/tools"
)

// Shell runs one shell command, streaming its output to output. A nil error
// means exit status zero.
type Shell func(ctx context.Context, command string, output io.Writer) error

// Options is everything Run needs from the outside world.
type Options struct {
	Files            fs.FS
	Home             string
	Harness          string
	InstallTools     bool
	KeepInstructions bool
	Yes              bool
	Interactive      bool
	Stdin            io.Reader
	Stdout           io.Writer
	Shell            Shell
	Now              func() time.Time
}

type assets struct {
	skills   fs.FS
	letter   string
	tools    []tools.Tool
	vendored []string
}

type harnessPlan struct {
	harness harness.Harness
	skills  skills.Plan
	letter  letter.Change
}

type toolStatus struct {
	tool         tools.Tool
	present      bool
	skillPresent bool
	install      bool
}

type plan struct {
	harnesses []harnessPlan
	tools     []toolStatus
}

type pickSource int

const (
	fromDetect pickSource = iota
	fromConfig
	fromFlag
)

// Run prints the plan, asks what it has to, applies, and reports. Without a
// terminal it changes nothing unless Yes is set.
func Run(ctx context.Context, opts Options) error {
	embedded, err := loadAssets(opts.Files)
	if err != nil {
		return err
	}
	config, err := loadConfig(opts.Home)
	if err != nil {
		return err
	}
	picked, source, err := choose(opts, config)
	if err != nil {
		return err
	}
	out := opts.Stdout
	printHarnesses(out, opts.Home, picked)
	current, err := buildPlan(ctx, opts, embedded, picked)
	if err != nil {
		return err
	}
	printPlan(out, opts.Home, embedded, current)
	if !opts.Interactive && !opts.Yes {
		printRerun(out, opts, picked, current)
		return nil
	}
	if opts.Interactive && !opts.Yes {
		ask := prompt.New(opts.Stdin, out)
		if source == fromDetect {
			repicked, err := askHarnesses(ask, opts.Home, picked)
			if err != nil {
				return err
			}
			if strings.Join(harness.Keys(repicked), ",") != strings.Join(harness.Keys(picked), ",") {
				picked = repicked
				if current, err = replan(opts, embedded, picked, current); err != nil {
					return err
				}
				printPlan(out, opts.Home, embedded, current)
			}
		} else {
			apply, err := ask.Confirm(fmt.Sprintf("\nApply to %s?", names(picked)), true)
			if err != nil {
				return err
			}
			if !apply {
				fmt.Fprintln(out, "Nothing changed.")
				return nil
			}
		}
		if err := askTools(ask, opts, current); err != nil {
			return err
		}
	} else {
		for index := range current.tools {
			status := &current.tools[index]
			status.install = !status.present && opts.InstallTools && status.tool.Command != ""
		}
	}
	if len(picked) == 0 {
		fmt.Fprintln(out, "\nNo harness picked. Nothing changed. Pass --harness claude,codex to name one.")
		return nil
	}
	stamp := opts.Now().Format("20060102-150405")
	if err := applyHarnesses(opts, embedded, current, stamp); err != nil {
		return err
	}
	if err := saveConfig(opts.Home, Config{Harnesses: harness.Keys(picked)}); err != nil {
		return err
	}
	toolsErr := applyTools(ctx, opts, current)
	fmt.Fprintf(out, "\nharness picks saved to %s\nrestart the harness so the skills load.\n", display(opts.Home, configPath(opts.Home)))
	return toolsErr
}

func loadAssets(files fs.FS) (assets, error) {
	skillsFS, err := fs.Sub(files, "skills")
	if err != nil {
		return assets{}, fmt.Errorf("[JSTACK-EMBED] the binary has no skills folder embedded: %w; reinstall it", err)
	}
	letterText, err := fs.ReadFile(files, "AGENTS.md")
	if err != nil {
		return assets{}, fmt.Errorf("[JSTACK-EMBED] the binary has no AGENTS.md embedded: %w; reinstall it", err)
	}
	toolsText, err := fs.ReadFile(files, "tools.md")
	if err != nil {
		return assets{}, fmt.Errorf("[JSTACK-EMBED] the binary has no tools.md embedded: %w; reinstall it", err)
	}
	vendorText, err := fs.ReadFile(files, "vendor.json")
	if err != nil {
		return assets{}, fmt.Errorf("[JSTACK-EMBED] the binary has no vendor.json embedded: %w; reinstall it", err)
	}
	var vendor struct {
		Skills []struct {
			Name string `json:"name"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(vendorText, &vendor); err != nil {
		return assets{}, fmt.Errorf("[JSTACK-EMBED] the embedded vendor.json is not valid JSON: %w; rebuild the binary from a checkout where make verify passes", err)
	}
	var vendored []string
	for _, entry := range vendor.Skills {
		vendored = append(vendored, entry.Name)
	}
	return assets{skills: skillsFS, letter: string(letterText), tools: tools.Parse(string(toolsText)), vendored: vendored}, nil
}

func choose(opts Options, config Config) ([]harness.Harness, pickSource, error) {
	if opts.Harness != "" {
		picked, err := harness.Parse(opts.Harness)
		return picked, fromFlag, err
	}
	if len(config.Harnesses) > 0 {
		picked, err := harness.ByKeys(config.Harnesses)
		if err != nil {
			return nil, fromConfig, fmt.Errorf("%w; the saved picks in %q are stale, pass --harness to replace them", err, configPath(opts.Home))
		}
		return picked, fromConfig, nil
	}
	return harness.Found(opts.Home), fromDetect, nil
}

func buildPlan(ctx context.Context, opts Options, embedded assets, picked []harness.Harness) (plan, error) {
	harnessPlans, err := planHarnesses(opts, embedded, picked)
	if err != nil {
		return plan{}, err
	}
	current := plan{harnesses: harnessPlans}
	for _, tool := range embedded.tools {
		current.tools = append(current.tools, toolStatus{tool: tool, present: opts.Shell(ctx, tool.Check, io.Discard) == nil})
	}
	markSkillPresence(opts.Home, picked, current.tools)
	return current, nil
}

// replan redoes the harness half after a new pick. The tool checks already ran
// and their answers don't depend on the pick, only the skill presence does.
func replan(opts Options, embedded assets, picked []harness.Harness, current plan) (plan, error) {
	harnessPlans, err := planHarnesses(opts, embedded, picked)
	if err != nil {
		return plan{}, err
	}
	current.harnesses = harnessPlans
	markSkillPresence(opts.Home, picked, current.tools)
	return current, nil
}

func planHarnesses(opts Options, embedded assets, picked []harness.Harness) ([]harnessPlan, error) {
	var plans []harnessPlan
	for _, entry := range picked {
		skillPlan, err := skills.PlanFor(embedded.skills, entry.SkillsDir(opts.Home))
		if err != nil {
			return nil, err
		}
		path := entry.InstructionsPath(opts.Home)
		existing, err := os.ReadFile(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("[JSTACK-LETTER-READ] cannot read %q: %w; make it readable and rerun", path, err)
		}
		plans = append(plans, harnessPlan{
			harness: entry,
			skills:  skillPlan,
			letter:  letter.Plan(string(existing), embedded.letter, entry.Lead, opts.KeepInstructions),
		})
	}
	return plans, nil
}

func markSkillPresence(home string, picked []harness.Harness, statuses []toolStatus) {
	for index := range statuses {
		if folder := statuses[index].tool.SkillFolder; folder != "" {
			statuses[index].skillPresent = skillPresent(home, picked, folder)
		}
	}
}

func skillPresent(home string, picked []harness.Harness, folder string) bool {
	if isDir(filepath.Join(home, ".agents", "skills", folder)) {
		return true
	}
	for _, entry := range picked {
		if !isDir(filepath.Join(entry.SkillsDir(home), folder)) {
			return false
		}
	}
	return true
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func printHarnesses(out io.Writer, home string, picked []harness.Harness) {
	fmt.Fprintln(out, "harnesses")
	for _, entry := range harness.All() {
		mark, state := " ", "not found"
		if entry.Installed(home) {
			state = "found"
		}
		if contains(picked, entry) {
			mark = "x"
		}
		fmt.Fprintf(out, "  [%s] %-11s %s, %s\n", mark, entry.Name, display(home, entry.DetectDir(home)), state)
	}
}

func printPlan(out io.Writer, home string, embedded assets, current plan) {
	for _, entry := range current.harnesses {
		fmt.Fprintf(out, "\n%s  %s\n", entry.harness.Name, display(home, entry.harness.SkillsDir(home)))
		fmt.Fprintf(out, "  new      %s\n", list(entry.skills.New))
		fmt.Fprintf(out, "  changed  %s\n", list(entry.skills.Changed))
		fmt.Fprintf(out, "  same     %d skills\n", len(entry.skills.Same))
		fmt.Fprintf(out, "  local    %s (untouched)\n", list(entry.skills.Local))
		fmt.Fprintf(out, "  letter   %s %s\n", display(home, entry.harness.InstructionsPath(home)), letterIntent(entry.letter.Outcome))
	}
	if len(embedded.vendored) > 0 {
		fmt.Fprintf(out, "\nvendored %s (upstream text, pinned in vendor.json)\n", list(embedded.vendored))
	}
	fmt.Fprintln(out, "\ntools")
	for _, status := range current.tools {
		fmt.Fprintf(out, "  %s\n", toolIntent(status))
	}
}

func letterIntent(outcome letter.Outcome) string {
	switch outcome {
	case letter.Create:
		return "would be created"
	case letter.Update:
		return "would get the new letter between the markers"
	case letter.Replace:
		return "has other content: would be replaced by the letter and backed up"
	case letter.Append:
		return "would get the letter appended"
	}
	return "up to date"
}

func toolIntent(status toolStatus) string {
	if !status.present {
		return fmt.Sprintf("missing %s. install: %s", status.tool.Title, status.tool.Install)
	}
	line := "ok " + status.tool.Title
	if status.tool.SkillInstall == "" || status.tool.SkillFolder == "" {
		return line
	}
	if status.skillPresent {
		return line + ", skill present"
	}
	return line + ", skill missing, would run: " + status.tool.SkillInstall
}

func printRerun(out io.Writer, opts Options, picked []harness.Harness, current plan) {
	fmt.Fprintln(out, "\nNo terminal, so nothing changed. Rerun with the flags to apply:")
	keys := strings.Join(harness.Keys(picked), ",")
	if keys == "" {
		keys = "claude,codex"
	}
	line := "jstack setup --harness " + keys + " --yes"
	if opts.InstallTools {
		line += " --install-tools"
	}
	if opts.KeepInstructions {
		line += " --keep-instructions"
	}
	fmt.Fprintf(out, "  %s\n", line)
	for _, status := range current.tools {
		if !opts.InstallTools && !status.present && status.tool.Command != "" {
			fmt.Fprintln(out, "  add --install-tools to also install the missing tools")
			break
		}
	}
	for _, entry := range current.harnesses {
		if !opts.KeepInstructions && entry.letter.Outcome == letter.Replace && entry.harness.Lead == "" {
			fmt.Fprintf(out, "  add --keep-instructions to append the letter to %s instead of replacing it\n", display(opts.Home, entry.harness.InstructionsPath(opts.Home)))
		}
	}
}

func askHarnesses(ask *prompt.Prompt, home string, picked []harness.Harness) ([]harness.Harness, error) {
	all := harness.All()
	labels := make([]string, 0, len(all))
	selected := make([]bool, 0, len(all))
	for _, entry := range all {
		state := "not found"
		if entry.Installed(home) {
			state = "found"
		}
		labels = append(labels, fmt.Sprintf("%-11s %s, %s", entry.Name, display(home, entry.DetectDir(home)), state))
		selected = append(selected, contains(picked, entry))
	}
	chosen, err := ask.MultiSelect("Install into which harnesses?", labels, selected)
	if err != nil {
		return nil, err
	}
	var keys []string
	for index, entry := range all {
		if chosen[index] {
			keys = append(keys, entry.Key)
		}
	}
	return harness.ByKeys(keys)
}

func askTools(ask *prompt.Prompt, opts Options, current plan) error {
	for index := range current.tools {
		status := &current.tools[index]
		if status.present || status.tool.Command == "" {
			continue
		}
		if opts.InstallTools {
			status.install = true
			continue
		}
		install, err := ask.Confirm(fmt.Sprintf("\nInstall %s? (%s)", status.tool.Title, status.tool.Command), false)
		if err != nil {
			return err
		}
		status.install = install
	}
	return nil
}

func applyHarnesses(opts Options, embedded assets, current plan, stamp string) error {
	out := opts.Stdout
	for _, entry := range current.harnesses {
		backup := filepath.Join(opts.Home, ".jstack", "backup", stamp, entry.harness.Key)
		fmt.Fprintf(out, "\n%s\n", entry.harness.Name)
		if err := applySkills(embedded, entry, opts.Home, backup, out); err != nil {
			return err
		}
		if err := applyLetter(entry, opts.Home, backup, out); err != nil {
			return err
		}
	}
	return nil
}

// applyTools installs what was agreed and reports every tool. The skills and
// the letter are already in place, so a tool that fails is reported at the end
// instead of stopping the run.
func applyTools(ctx context.Context, opts Options, current plan) error {
	fmt.Fprintln(opts.Stdout, "\ntools")
	var failures []error
	for _, status := range current.tools {
		if err := applyTool(ctx, opts, status, opts.Stdout); err != nil {
			failures = append(failures, err)
		}
	}
	if len(failures) == 0 {
		return nil
	}
	return fmt.Errorf("[JSTACK-TOOLS] the skills and the letter are in place, but %d tool step(s) failed:\n%w", len(failures), errors.Join(failures...))
}

func applySkills(embedded assets, entry harnessPlan, home, backup string, out io.Writer) error {
	dest := entry.harness.SkillsDir(home)
	if !entry.skills.Pending() {
		fmt.Fprintf(out, "  skills   up to date in %s\n", display(home, dest))
		return nil
	}
	skillsBackup := filepath.Join(backup, "skills")
	if err := skills.Apply(embedded.skills, dest, entry.skills, skillsBackup); err != nil {
		return err
	}
	fmt.Fprintf(out, "  skills   %d installed, %d updated in %s\n", len(entry.skills.New), len(entry.skills.Changed), display(home, dest))
	if len(entry.skills.Changed) > 0 {
		fmt.Fprintf(out, "  backup   %s\n", display(home, skillsBackup))
	}
	after, err := skills.PlanFor(embedded.skills, dest)
	if err != nil {
		return err
	}
	if after.Pending() {
		fmt.Fprintf(out, "  remaining drift: %s\n", list(append(after.New, after.Changed...)))
	}
	return nil
}

func applyLetter(entry harnessPlan, home, backup string, out io.Writer) error {
	path := entry.harness.InstructionsPath(home)
	shown := display(home, path)
	if entry.letter.Outcome == letter.Same {
		fmt.Fprintf(out, "  letter   up to date in %s\n", shown)
		return nil
	}
	if entry.letter.Outcome == letter.Replace {
		saved := filepath.Join(backup, filepath.Base(path))
		if err := copyFile(path, saved); err != nil {
			return fmt.Errorf("[JSTACK-LETTER-BACKUP] cannot back up %q to %q: %w; the file was not changed, fix the permissions and rerun", path, saved, err)
		}
		fmt.Fprintf(out, "  letter   replaced %s, old file backed up to %s\n", shown, display(home, saved))
	} else {
		fmt.Fprintf(out, "  letter   %s %s\n", letterPast(entry.letter.Outcome), shown)
	}
	return writeFile(path, entry.letter.Content)
}

func letterPast(outcome letter.Outcome) string {
	switch outcome {
	case letter.Create:
		return "created"
	case letter.Append:
		return "appended to"
	}
	return "updated between the markers in"
}

func applyTool(ctx context.Context, opts Options, status toolStatus, out io.Writer) error {
	if !status.present {
		if !status.install {
			fmt.Fprintf(out, "  missing %s. install: %s\n", status.tool.Title, status.tool.Install)
			return nil
		}
		fmt.Fprintf(out, "  installing %s: %s\n", status.tool.Title, status.tool.Command)
		if err := opts.Shell(ctx, status.tool.Command, out); err != nil {
			fmt.Fprintf(out, "  FAILED %s: %v\n", status.tool.Title, err)
			return fmt.Errorf("%s: `%s` failed: %v; run it by hand, then rerun jstack setup", status.tool.Title, status.tool.Command, err)
		}
		if err := opts.Shell(ctx, status.tool.Check, io.Discard); err != nil {
			fmt.Fprintf(out, "  FAILED %s: installed, but its check still fails\n", status.tool.Title)
			return fmt.Errorf("%s: `%s` ran, but the check `%s` still fails; finish the install line by hand (%s), then rerun jstack setup", status.tool.Title, status.tool.Command, status.tool.Check, status.tool.Install)
		}
		fmt.Fprintf(out, "  installed %s\n", status.tool.Title)
	}
	line := "  ok " + status.tool.Title
	if status.tool.SkillInstall == "" || status.tool.SkillFolder == "" {
		fmt.Fprintln(out, line)
		return nil
	}
	if status.skillPresent {
		fmt.Fprintln(out, line+", skill present")
		return nil
	}
	if err := opts.Shell(ctx, status.tool.SkillInstall, io.Discard); err != nil {
		fmt.Fprintf(out, "%s, skill install FAILED via %s: %v\n", line, status.tool.SkillInstall, err)
		return fmt.Errorf("%s: `%s` failed: %v; run it by hand so the tool's skill is in place", status.tool.Title, status.tool.SkillInstall, err)
	}
	fmt.Fprintf(out, "%s, skill installed via %s\n", line, status.tool.SkillInstall)
	return nil
}

func copyFile(source, destination string) error {
	content, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	return os.WriteFile(destination, content, 0o600)
}

// writeFile replaces path in one step: the new content lands in a temporary
// file beside it and is renamed over the original only once fully written, so
// a failed write leaves the user's file as it was. A symlink, the dotfiles
// repo case, stays a symlink: the file it points to is what gets replaced.
func writeFile(path, content string) error {
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
		if resolved, err := filepath.EvalSymlinks(path); err == nil {
			path = resolved
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("[JSTACK-LETTER-WRITE] cannot create %q: %w; make the parent writable and rerun", filepath.Dir(path), err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".jstack-letter-*")
	if err != nil {
		return fmt.Errorf("[JSTACK-LETTER-WRITE] cannot stage a new %q: %w; make its folder writable and rerun", path, err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	fail := func(step string, cause error) error {
		_ = temporary.Close()
		return fmt.Errorf("[JSTACK-LETTER-WRITE] cannot %s for %q: %w; the file was not changed, fix the folder and rerun", step, path, cause)
	}
	if err := temporary.Chmod(mode); err != nil {
		return fail("set permissions", err)
	}
	if _, err := temporary.WriteString(content); err != nil {
		return fail("write the new content", err)
	}
	if err := temporary.Sync(); err != nil {
		return fail("sync the new content", err)
	}
	if err := temporary.Close(); err != nil {
		return fail("close the staged file", err)
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return fmt.Errorf("[JSTACK-LETTER-WRITE] cannot replace %q: %w; the file was not changed, fix the permissions and rerun", path, err)
	}
	return nil
}

func display(home, path string) string {
	if rest, ok := strings.CutPrefix(path, home+string(filepath.Separator)); ok {
		return "~/" + filepath.ToSlash(rest)
	}
	return path
}

func list(items []string) string {
	if len(items) == 0 {
		return "-"
	}
	return strings.Join(items, ", ")
}

func names(picked []harness.Harness) string {
	parts := make([]string, 0, len(picked))
	for _, entry := range picked {
		parts = append(parts, entry.Name)
	}
	return strings.Join(parts, ", ")
}

func contains(picked []harness.Harness, entry harness.Harness) bool {
	for _, candidate := range picked {
		if candidate.Key == entry.Key {
			return true
		}
	}
	return false
}
