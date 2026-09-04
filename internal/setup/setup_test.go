package setup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/janiorvalle/jstack/internal/harness"
	"github.com/janiorvalle/jstack/internal/letter"
	"github.com/janiorvalle/jstack/internal/skills"
	"github.com/janiorvalle/jstack/internal/tools"
)

const toolsFixture = "# Tools\n\n" +
	"## git\n\n- Check: `check-git`\n- macOS: `brew install git`\n\n" +
	"## roast\n\n- Repo: https://github.com/x/roast\n- Check: `check-roast`\n- Version: `version-roast`\n- Install: `curl roast | sh`\n- Skill install: `roast install-skill`\n- Skill folder: `roast`\n\n" +
	"## prose only\n\nno check line\n"

const pinnedInstall = "npm install -g browser@0.36.0 && browser install"

// pinnedFixture adds a tool whose install line pins a version, the way
// agent-browser is pinned in tools.md.
func pinnedFixture() fstest.MapFS {
	files := fixture()
	files["tools.md"] = &fstest.MapFile{Data: []byte(toolsFixture +
		"\n## browser\n\n- Repo: https://github.com/x/browser\n- Check: `check-browser`\n- Version: `version-browser`\n- Install: `" + pinnedInstall + "`\n")}
	return files
}

func fixture() fstest.MapFS {
	return fstest.MapFS{
		"skills/voice/SKILL.md": {Data: []byte("voice v2\n")},
		"skills/how/SKILL.md":   {Data: []byte("how\n")},
		"AGENTS.md":             {Data: []byte("# the letter\n")},
		"tools.md":              {Data: []byte(toolsFixture)},
		"vendor.json":           {Data: []byte(`{"skills":[{"name":"how"}]}`)},
	}
}

// fakeShell is the machine and the network: which checks pass, what the
// version commands print, and what each tool's source says is latest. A tool
// missing from latest is one whose lookup failed. The roast install line
// lands 1.1.0, the way a real install line installs the newest release,
// unless stuck is set: then an older roast keeps winning on PATH.
type fakeShell struct {
	present  map[string]bool
	failing  map[string]bool
	versions map[string]string
	latest   map[string]string
	stuck    bool
	commands []string
}

func withRoast(version string) *fakeShell {
	return &fakeShell{
		present:  map[string]bool{"check-git": true, "check-roast": true},
		versions: map[string]string{"version-roast": "roast " + version},
		latest:   map[string]string{"roast": "v1.1.0"},
	}
}

func (f *fakeShell) run(_ context.Context, command string, out io.Writer) error {
	f.commands = append(f.commands, command)
	if f.failing[command] {
		return errors.New("exit status 1")
	}
	if strings.HasPrefix(command, "check-") && !f.present[command] {
		return errors.New("exit status 1")
	}
	if command == "curl roast | sh" {
		if f.present == nil {
			f.present = map[string]bool{}
		}
		if f.versions == nil {
			f.versions = map[string]string{}
		}
		f.present["check-roast"] = true
		if !f.stuck {
			f.versions["version-roast"] = "roast 1.1.0"
		}
	}
	if command == pinnedInstall && !f.stuck {
		f.versions["version-browser"] = "browser 0.36.0"
	}
	if output, ok := f.versions[command]; ok {
		fmt.Fprintln(out, output)
	}
	return nil
}

func (f *fakeShell) lookup(_ context.Context, tool tools.Tool) (string, error) {
	if latest, ok := f.latest[tool.Title]; ok {
		return latest, nil
	}
	return "", errors.New("dial tcp: connection refused")
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func options(t *testing.T, home string, shell *fakeShell, stdin string) (Options, *bytes.Buffer) {
	t.Helper()
	var out bytes.Buffer
	return Options{
		Files:  fixture(),
		Home:   home,
		Getenv: func(string) string { return "" },
		Stdin:  strings.NewReader(stdin),
		Stdout: &out,
		Shell:  shell.run,
		Latest: shell.lookup,
		Now:    func() time.Time { return time.Date(2026, 9, 3, 10, 4, 5, 0, time.UTC) },
	}, &out
}

func homeWithClaude(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	write(t, filepath.Join(home, ".claude", "CLAUDE.md"), "# my own notes\n")
	write(t, filepath.Join(home, ".claude", "skills", "voice", "SKILL.md"), "voice v1\n")
	write(t, filepath.Join(home, ".claude", "skills", "mine", "SKILL.md"), "local\n")
	return home
}

func TestNoTerminalPrintsPlanAndRerunFlagsAndChangesNothing(t *testing.T) {
	home := homeWithClaude(t)
	shell := &fakeShell{present: map[string]bool{"check-git": true}, latest: map[string]string{"roast": "v1.1.0"}}
	opts, out := options(t, home, shell, "")
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"[x] Claude Code ~/.claude, found",
		"[ ] Codex       ~/.codex, not found",
		"new      how",
		"changed  voice",
		"local    mine (untouched)",
		"~/.claude/CLAUDE.md has other content: would be replaced by the letter and backed up",
		"vendored how",
		"ok git",
		"missing roast. install: curl roast | sh",
		"jstack setup --harness claude --yes",
		"add --install-tools to also install the missing tools",
		"add --keep-instructions to append the letter to ~/.claude/CLAUDE.md instead of replacing it",
	} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("output missing %q:\n%s", expected, out.String())
		}
	}
	if read(t, filepath.Join(home, ".claude", "CLAUDE.md")) != "# my own notes\n" {
		t.Fatal("instructions file changed")
	}
	if read(t, filepath.Join(home, ".claude", "skills", "voice", "SKILL.md")) != "voice v1\n" {
		t.Fatal("skill changed")
	}
	if exists(filepath.Join(home, ".jstack")) {
		t.Fatal("~/.jstack was created")
	}
	if strings.Join(shell.commands, ";") != "check-git;check-roast" {
		t.Fatalf("commands = %v", shell.commands)
	}
}

func TestYesAppliesBacksUpAndSavesPicks(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRoast("1.1.0")
	opts, out := options(t, home, shell, "")
	opts.Yes = true
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(home, ".jstack", "backup", "20260903-100405", "claude")
	if got := read(t, filepath.Join(home, ".claude", "skills", "voice", "SKILL.md")); got != "voice v2\n" {
		t.Fatalf("voice = %q", got)
	}
	if got := read(t, filepath.Join(backup, "skills", "voice", "SKILL.md")); got != "voice v1\n" {
		t.Fatalf("voice backup = %q", got)
	}
	if got := read(t, filepath.Join(home, ".claude", "skills", "how", "SKILL.md")); got != "how\n" {
		t.Fatalf("how = %q", got)
	}
	if got := read(t, filepath.Join(home, ".claude", "skills", "mine", "SKILL.md")); got != "local\n" {
		t.Fatalf("mine = %q", got)
	}
	if got := read(t, filepath.Join(home, ".claude", "CLAUDE.md")); got != letter.Block("# the letter\n") {
		t.Fatalf("CLAUDE.md = %q", got)
	}
	if got := read(t, filepath.Join(backup, "CLAUDE.md")); got != "# my own notes\n" {
		t.Fatalf("CLAUDE.md backup = %q", got)
	}
	if got := read(t, filepath.Join(home, ".jstack", "config.json")); got != "{\n  \"harnesses\": [\n    \"claude\"\n  ]\n}\n" {
		t.Fatalf("config = %q", got)
	}
	if strings.Join(shell.commands, ";") != "check-git;check-roast;version-roast;roast install-skill" {
		t.Fatalf("commands = %v", shell.commands)
	}
	for _, expected := range []string{
		"skills   1 installed, 1 updated in ~/.claude/skills",
		"backup   ~/.jstack/backup/20260903-100405/claude/skills",
		"letter   replaced ~/.claude/CLAUDE.md, old file backed up to ~/.jstack/backup/20260903-100405/claude/CLAUDE.md",
		"ok roast 1.1.0, skill installed via roast install-skill",
		"restart the harness so the skills load",
	} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("output missing %q:\n%s", expected, out.String())
		}
	}
}

func TestVariableMovesTheRowThroughPlanAndApply(t *testing.T) {
	home := homeWithClaude(t)
	codexHome := filepath.Join(t.TempDir(), "codex")
	write(t, filepath.Join(codexHome, "config.toml"), "")
	shell := &fakeShell{present: map[string]bool{"check-git": true, "check-roast": true}}
	opts, out := options(t, home, shell, "")
	opts.Getenv = func(name string) string {
		if name == "CODEX_HOME" {
			return codexHome
		}
		return ""
	}
	opts.Yes = true
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"[x] Claude Code ~/.claude, found",
		"[x] Codex       " + codexHome + " (CODEX_HOME), found",
		"Codex  " + filepath.Join(codexHome, "skills"),
		"skills   2 installed, 0 updated in " + filepath.Join(codexHome, "skills"),
		"letter   created " + filepath.Join(codexHome, "AGENTS.md"),
	} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("output missing %q:\n%s", expected, out.String())
		}
	}
	if got := read(t, filepath.Join(codexHome, "skills", "voice", "SKILL.md")); got != "voice v2\n" {
		t.Fatalf("voice = %q", got)
	}
	if got := read(t, filepath.Join(codexHome, "AGENTS.md")); got != letter.Block("# the letter\n") {
		t.Fatalf("AGENTS.md = %q", got)
	}
	if exists(filepath.Join(home, ".codex")) {
		t.Fatal("~/.codex was created although CODEX_HOME points elsewhere")
	}
	if got := read(t, filepath.Join(home, ".jstack", "config.json")); got != "{\n  \"harnesses\": [\n    \"claude\",\n    \"codex\"\n  ]\n}\n" {
		t.Fatalf("config = %q", got)
	}
}

func TestSecondRunUsesSavedPicksAndReportsUpToDate(t *testing.T) {
	home := homeWithClaude(t)
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	shell := withRoast("1.1.0")
	opts, _ := options(t, home, shell, "")
	opts.Yes = true
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(home, ".claude", "skills", "roast", "SKILL.md"), "roast\n")
	write(t, filepath.Join(home, ".codex", "skills", "roast", "SKILL.md"), "roast\n")
	shell.commands = nil
	opts, out := options(t, home, shell, "\n")
	opts.Interactive = true
	opts.Now = func() time.Time { return time.Date(2026, 9, 3, 11, 0, 0, 0, time.UTC) }
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "Install into which harnesses?") {
		t.Fatalf("asked for harnesses again:\n%s", out.String())
	}
	for _, expected := range []string{"Apply to Claude Code, Codex? [Y/n]", "skills   up to date in ~/.claude/skills", "letter   up to date in ~/.claude/CLAUDE.md", "ok roast 1.1.0, skill present"} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("output missing %q:\n%s", expected, out.String())
		}
	}
	if exists(filepath.Join(home, ".jstack", "backup", "20260903-110000")) {
		t.Fatal("a backup was made with nothing changed")
	}
}

func TestTerminalAsksHarnessesThenToolsThenApplies(t *testing.T) {
	home := homeWithClaude(t)
	shell := &fakeShell{present: map[string]bool{"check-git": true}, latest: map[string]string{"roast": "v1.1.0"}}
	opts, out := options(t, home, shell, "3\n\ny\n")
	opts.Interactive = true
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"Install into which harnesses?",
		"1. [x] Claude Code",
		"3. [ ] OpenCode",
		"OpenCode  ~/.config/opencode/skills",
		"Install roast? (curl roast | sh) [y/N]",
		"installing roast: curl roast | sh",
		"installed roast 1.1.0",
		"ok roast 1.1.0, skill installed via roast install-skill",
	} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("output missing %q:\n%s", expected, out.String())
		}
	}
	if !exists(filepath.Join(home, ".config", "opencode", "skills", "voice", "SKILL.md")) || !exists(filepath.Join(home, ".config", "opencode", "AGENTS.md")) {
		t.Fatal("OpenCode did not get the skills and the letter")
	}
	if got := read(t, filepath.Join(home, ".jstack", "config.json")); !strings.Contains(got, `"claude",`) || !strings.Contains(got, `"opencode"`) {
		t.Fatalf("config = %q", got)
	}
	if strings.Join(shell.commands, ";") != "check-git;check-roast;curl roast | sh;check-roast;version-roast;roast install-skill" {
		t.Fatalf("commands = %v", shell.commands)
	}
}

func TestOutdatedToolIsReportedWithBothVersionsAndTheRerunHint(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRoast("1.0.0")
	opts, out := options(t, home, shell, "")
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"outdated roast 1.0.0, latest 1.1.0. update: curl roast | sh",
		"add --update-tools to also update the outdated tools",
	} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("output missing %q:\n%s", expected, out.String())
		}
	}
	if strings.Contains(out.String(), "add --install-tools") {
		t.Fatalf("hint to install with nothing missing:\n%s", out.String())
	}
	if strings.Join(shell.commands, ";") != "check-git;check-roast;version-roast" {
		t.Fatalf("commands = %v", shell.commands)
	}
}

func TestPinnedToolIsComparedWithThePinNotTheRegistry(t *testing.T) {
	for installed, expected := range map[string]string{
		"0.27.0": "outdated browser 0.27.0, pinned 0.36.0. update: " + pinnedInstall,
		"0.36.0": "ok browser 0.36.0",
		"0.37.0": "ahead browser 0.37.0, pinned 0.36.0. update: " + pinnedInstall,
		"":       "outdated browser version unknown, pinned 0.36.0. update: " + pinnedInstall,
	} {
		home := homeWithClaude(t)
		shell := withRoast("1.1.0")
		shell.present["check-browser"] = true
		shell.versions["version-browser"] = "browser " + installed
		opts, out := options(t, home, shell, "")
		opts.Files = pinnedFixture()
		if err := Run(context.Background(), opts); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), expected+"\n") {
			t.Errorf("installed %s: output missing %q:\n%s", installed, expected, out.String())
		}
	}
}

func TestUpdateOfAPinnedToolAheadOfThePinInstallsThePin(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRoast("1.1.0")
	shell.present["check-browser"] = true
	shell.versions["version-browser"] = "browser 0.37.0"
	opts, out := options(t, home, shell, "")
	opts.Files = pinnedFixture()
	opts.Yes = true
	opts.UpdateTools = true
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"updating browser 0.37.0 to 0.36.0: " + pinnedInstall, "updated browser 0.36.0"} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("output missing %q:\n%s", expected, out.String())
		}
	}
}

func TestUpdateOfAPinnedToolThatStillPrintsNoVersionIsAnError(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRoast("1.1.0")
	shell.present["check-browser"] = true
	shell.versions["version-browser"] = "usage: browser [command]"
	shell.stuck = true
	opts, out := options(t, home, shell, "")
	opts.Files = pinnedFixture()
	opts.Yes = true
	opts.UpdateTools = true
	err := Run(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "browser: `"+pinnedInstall+"` ran, but `version-browser` prints no version, so the pinned 0.36.0 can't be confirmed") {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(out.String(), "updating browser version unknown to 0.36.0: "+pinnedInstall) || !strings.Contains(out.String(), "FAILED browser: updated, but `version-browser` prints no version") {
		t.Fatalf("output:\n%s", out.String())
	}
}

func TestTerminalYesToUpdateRunsTheInstallLineAndReinstallsTheSkill(t *testing.T) {
	home := homeWithClaude(t)
	write(t, filepath.Join(home, ".claude", "skills", "roast", "SKILL.md"), "roast 1.0.0\n")
	shell := withRoast("1.0.0")
	opts, out := options(t, home, shell, "\ny\n")
	opts.Interactive = true
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"Update roast 1.0.0 to 1.1.0? (curl roast | sh) [y/N]",
		"updating roast 1.0.0 to 1.1.0: curl roast | sh",
		"updated roast 1.1.0",
		"ok roast 1.1.0, skill installed via roast install-skill",
	} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("output missing %q:\n%s", expected, out.String())
		}
	}
	if strings.Join(shell.commands, ";") != "check-git;check-roast;version-roast;curl roast | sh;check-roast;version-roast;roast install-skill" {
		t.Fatalf("commands = %v", shell.commands)
	}
}

func TestTerminalNoToUpdateLeavesItOutdated(t *testing.T) {
	home := homeWithClaude(t)
	write(t, filepath.Join(home, ".claude", "skills", "roast", "SKILL.md"), "roast 1.0.0\n")
	shell := withRoast("1.0.0")
	opts, out := options(t, home, shell, "\nn\n")
	opts.Interactive = true
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "outdated roast 1.0.0, latest 1.1.0. update: curl roast | sh, skill present") {
		t.Fatalf("output:\n%s", out.String())
	}
	if strings.Join(shell.commands, ";") != "check-git;check-roast;version-roast" {
		t.Fatalf("commands = %v", shell.commands)
	}
}

func TestUpdateToolsWithYesUpdatesTheOutdatedTools(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRoast("1.0.0")
	opts, out := options(t, home, shell, "")
	opts.Yes = true
	opts.UpdateTools = true
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "updated roast 1.1.0") {
		t.Fatalf("output:\n%s", out.String())
	}
	if strings.Join(shell.commands, ";") != "check-git;check-roast;version-roast;curl roast | sh;check-roast;version-roast;roast install-skill" {
		t.Fatalf("commands = %v", shell.commands)
	}
}

func TestInstallToolsAloneDoesNotUpdate(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRoast("1.0.0")
	opts, out := options(t, home, shell, "")
	opts.Yes = true
	opts.InstallTools = true
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "outdated roast 1.0.0, latest 1.1.0") || strings.Contains(out.String(), "updating") {
		t.Fatalf("output:\n%s", out.String())
	}
}

func TestLatestLookupFailureReportsUnknownAndStillApplies(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRoast("1.0.0")
	shell.latest = nil
	opts, out := options(t, home, shell, "")
	opts.Yes = true
	opts.UpdateTools = true
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "ok roast 1.0.0, latest unknown, skill installed via roast install-skill") {
		t.Fatalf("output:\n%s", out.String())
	}
	if strings.Contains(out.String(), "outdated") || strings.Contains(out.String(), "updating") {
		t.Fatalf("offered an update with latest unknown:\n%s", out.String())
	}
	if !exists(filepath.Join(home, ".jstack", "config.json")) {
		t.Fatal("setup did not complete")
	}
}

func TestVersionCommandWithNoVersionReportsUnknown(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRoast("1.0.0")
	shell.versions["version-roast"] = "usage: roast [command]"
	opts, out := options(t, home, shell, "")
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "ok roast, version unknown, skill missing") || strings.Contains(out.String(), "--update-tools") {
		t.Fatalf("output:\n%s", out.String())
	}
}

func TestFailedUpdateIsAnErrorAfterTheRestLanded(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRoast("1.0.0")
	shell.failing = map[string]bool{"curl roast | sh": true}
	opts, out := options(t, home, shell, "")
	opts.Yes = true
	opts.UpdateTools = true
	err := Run(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "JSTACK-TOOLS") || !strings.Contains(err.Error(), "roast: `curl roast | sh` failed") {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(out.String(), "FAILED roast") || !exists(filepath.Join(home, ".jstack", "config.json")) {
		t.Fatalf("output:\n%s", out.String())
	}
}

func TestUpdateThatLeavesTheOldVersionOnPathIsAnError(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRoast("1.0.0")
	shell.stuck = true
	opts, out := options(t, home, shell, "")
	opts.Yes = true
	opts.UpdateTools = true
	err := Run(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "roast: `curl roast | sh` ran, but `version-roast` still prints 1.0.0 and the latest is 1.1.0") || !strings.Contains(err.Error(), "PATH") {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(out.String(), "FAILED roast: updated, but `version-roast` still prints 1.0.0") || strings.Contains(out.String(), "updated roast 1") {
		t.Fatalf("output:\n%s", out.String())
	}
	if strings.Join(shell.commands, ";") != "check-git;check-roast;version-roast;curl roast | sh;check-roast;version-roast" {
		t.Fatalf("commands = %v", shell.commands)
	}
}

func TestTerminalNoToToolKeepsItMissing(t *testing.T) {
	home := homeWithClaude(t)
	shell := &fakeShell{present: map[string]bool{"check-git": true}, latest: map[string]string{"roast": "v1.1.0"}}
	opts, out := options(t, home, shell, "\nn\n")
	opts.Interactive = true
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "missing roast. install: curl roast | sh") {
		t.Fatalf("output:\n%s", out.String())
	}
	if strings.Join(shell.commands, ";") != "check-git;check-roast" {
		t.Fatalf("commands = %v", shell.commands)
	}
}

func TestHarnessFlagOverridesSavedPicks(t *testing.T) {
	home := homeWithClaude(t)
	write(t, filepath.Join(home, ".jstack", "config.json"), `{"harnesses":["claude"]}`)
	shell := withRoast("1.1.0")
	opts, out := options(t, home, shell, "")
	opts.Yes = true
	opts.Harness = "pi"
	opts.InstallTools = true
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if !exists(filepath.Join(home, ".pi", "agent", "skills", "how", "SKILL.md")) || !exists(filepath.Join(home, ".pi", "agent", "AGENTS.md")) {
		t.Fatalf("pi did not get the files:\n%s", out.String())
	}
	if read(t, filepath.Join(home, ".claude", "skills", "voice", "SKILL.md")) != "voice v1\n" {
		t.Fatal("claude was touched")
	}
	if got := read(t, filepath.Join(home, ".jstack", "config.json")); !strings.Contains(got, `"pi"`) || strings.Contains(got, "claude") {
		t.Fatalf("config = %q", got)
	}
}

func TestInstallToolsWithYesInstallsMissingTools(t *testing.T) {
	home := homeWithClaude(t)
	shell := &fakeShell{present: map[string]bool{"check-git": true}, latest: map[string]string{"roast": "v1.1.0"}}
	opts, _ := options(t, home, shell, "")
	opts.Yes = true
	opts.InstallTools = true
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if strings.Join(shell.commands, ";") != "check-git;check-roast;curl roast | sh;check-roast;version-roast;roast install-skill" {
		t.Fatalf("commands = %v", shell.commands)
	}
}

func TestKeepInstructionsAppendsTheLetter(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRoast("1.1.0")
	opts, out := options(t, home, shell, "")
	opts.Yes = true
	opts.KeepInstructions = true
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(home, ".claude", "CLAUDE.md")); got != "# my own notes\n\n"+letter.Block("# the letter\n") {
		t.Fatalf("CLAUDE.md = %q", got)
	}
	if !strings.Contains(out.String(), "letter   appended to ~/.claude/CLAUDE.md") {
		t.Fatalf("output:\n%s", out.String())
	}
	if exists(filepath.Join(home, ".jstack", "backup", "20260903-100405", "claude", "CLAUDE.md")) {
		t.Fatal("append made a backup")
	}
}

func TestCursorRuleGetsTheFrontmatter(t *testing.T) {
	home := t.TempDir()
	shell := withRoast("1.1.0")
	opts, _ := options(t, home, shell, "")
	opts.Yes = true
	opts.Harness = "cursor"
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	got := read(t, filepath.Join(home, ".cursor", "rules", "jstack.mdc"))
	if !strings.HasPrefix(got, "---\ndescription: jstack") || !strings.Contains(got, "alwaysApply: true\n---\n\n"+letter.Start) {
		t.Fatalf("jstack.mdc = %q", got)
	}
}

func TestStaleSavedPickNamesTheFix(t *testing.T) {
	home := homeWithClaude(t)
	write(t, filepath.Join(home, ".jstack", "config.json"), `{"harnesses":["emacs"]}`)
	opts, _ := options(t, home, &fakeShell{}, "")
	err := Run(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "JSTACK-HARNESS-UNKNOWN") || !strings.Contains(err.Error(), "--harness") {
		t.Fatalf("err = %v", err)
	}
}

func TestNothingFoundWithYesChangesNothing(t *testing.T) {
	home := t.TempDir()
	opts, out := options(t, home, &fakeShell{}, "")
	opts.Yes = true
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "No harness picked. Nothing changed.") || exists(filepath.Join(home, ".jstack")) {
		t.Fatalf("output:\n%s", out.String())
	}
}

func TestFailedToolInstallIsAnErrorAfterTheRestLanded(t *testing.T) {
	home := homeWithClaude(t)
	shell := &fakeShell{present: map[string]bool{"check-git": true}, latest: map[string]string{"roast": "v1.1.0"}, failing: map[string]bool{"curl roast | sh": true}}
	opts, out := options(t, home, shell, "")
	opts.Yes = true
	opts.InstallTools = true
	err := Run(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "JSTACK-TOOLS") || !strings.Contains(err.Error(), "roast: `curl roast | sh` failed") {
		t.Fatalf("err = %v", err)
	}
	if !exists(filepath.Join(home, ".claude", "skills", "how", "SKILL.md")) || !exists(filepath.Join(home, ".jstack", "config.json")) {
		t.Fatal("skills or config missing after a tool failure")
	}
	if !strings.Contains(out.String(), "FAILED roast") || !strings.Contains(out.String(), "restart the harness") {
		t.Fatalf("output:\n%s", out.String())
	}
}

func TestInstallThatLeavesTheCheckFailingIsAnError(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRoast("1.1.0")
	shell.failing = map[string]bool{"check-roast": true}
	opts, out := options(t, home, shell, "")
	opts.Yes = true
	opts.InstallTools = true
	err := Run(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "roast: `curl roast | sh` ran, but the check `check-roast` still fails") {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(out.String(), "FAILED roast: installed, but its check still fails") {
		t.Fatalf("output:\n%s", out.String())
	}
	if strings.Join(shell.commands, ";") != "check-git;check-roast;curl roast | sh;check-roast" {
		t.Fatalf("commands = %v", shell.commands)
	}
}

const missingGit = "missing git, which setup doesn't install. get it by hand, see https://github.com/janiorvalle/jstack/blob/main/tools.md#git"

func TestMissingPrerequisiteIsReportedWithoutAnInstallOffer(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRoast("1.1.0")
	shell.present["check-git"] = false
	opts, out := options(t, home, shell, "")
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), missingGit) {
		t.Fatalf("output:\n%s", out.String())
	}
	if strings.Contains(out.String(), "add --install-tools") {
		t.Fatalf("hint to install a tool setup cannot install:\n%s", out.String())
	}
}

func TestMissingPrerequisiteIsNeverAskedAboutOrInstalled(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRoast("1.1.0")
	shell.present["check-git"] = false
	opts, out := options(t, home, shell, "\n")
	opts.Interactive = true
	opts.InstallTools = true
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "Install git?") {
		t.Fatalf("asked to install a prerequisite:\n%s", out.String())
	}
	if strings.Count(out.String(), missingGit) != 2 {
		t.Fatalf("the plan and the report both name the prerequisite once:\n%s", out.String())
	}
	if strings.Join(shell.commands, ";") != "check-git;check-roast;version-roast;roast install-skill" {
		t.Fatalf("commands = %v", shell.commands)
	}
}

func TestHasSavedPicks(t *testing.T) {
	home := t.TempDir()
	if HasSavedPicks(home) {
		t.Fatal("picks reported before any run")
	}
	write(t, filepath.Join(home, ".jstack", "config.json"), `{"harnesses":[]}`)
	if HasSavedPicks(home) {
		t.Fatal("empty picks reported as saved")
	}
	write(t, filepath.Join(home, ".jstack", "config.json"), `{"harnesses":["codex"]}`)
	if !HasSavedPicks(home) {
		t.Fatal("saved picks not reported")
	}
}

func TestNoTerminalRerunLineCarriesTheFlagsGiven(t *testing.T) {
	home := homeWithClaude(t)
	shell := &fakeShell{present: map[string]bool{"check-git": true}, latest: map[string]string{"roast": "v1.1.0"}}
	opts, out := options(t, home, shell, "")
	opts.KeepInstructions = true
	opts.InstallTools = true
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "jstack setup --harness claude --yes --install-tools --keep-instructions") {
		t.Fatalf("output:\n%s", out.String())
	}
	if strings.Contains(out.String(), "add --") {
		t.Fatalf("hints for flags already given:\n%s", out.String())
	}
	if read(t, filepath.Join(home, ".claude", "CLAUDE.md")) != "# my own notes\n" {
		t.Fatal("instructions file changed")
	}
}

func TestLetterWriteKeepsModeAndLeavesNoStagedFile(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".claude", "CLAUDE.md")
	write(t, path, "# mine\n")
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(path, "new\n"); err != nil {
		t.Fatal(err)
	}
	if got := read(t, path); got != "new\n" {
		t.Fatalf("content = %q", got)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, %v", info.Mode().Perm(), err)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries = %v, %v", entries, err)
	}
}

func TestLetterWriteFollowsASymlinkedInstructionsFile(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, "dotfiles", "AGENTS.md")
	write(t, target, "# mine\n")
	link := filepath.Join(home, ".codex", "AGENTS.md")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := writeFile(link, "new\n"); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("link replaced by a file: %v, %v", info, err)
	}
	if got := read(t, target); got != "new\n" {
		t.Fatalf("target = %q", got)
	}
}

func TestLetterApplyReplansAFileChangedAfterThePlan(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".codex", "AGENTS.md")
	write(t, path, letter.Block("old letter"))
	shell := withRoast("1.1.0")
	opts, out := options(t, home, shell, "")
	embedded, err := loadAssets(opts.Files)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := harness.Resolve(home, opts.Getenv).ByKeys([]string{"codex"})
	if err != nil {
		t.Fatal(err)
	}
	plans, err := planHarnesses(opts, embedded, rows)
	if err != nil {
		t.Fatal(err)
	}
	write(t, path, "# added while setup was asking\n"+letter.Block("old letter"))
	if err := applyLetter(opts, embedded, plans[0], filepath.Join(home, ".jstack", "backup", "x", "codex")); err != nil {
		t.Fatal(err)
	}
	if got := read(t, path); got != "# added while setup was asking\n"+letter.Block("# the letter\n") {
		t.Fatalf("file = %q", got)
	}
	if !strings.Contains(out.String(), "changed since the plan, planned again") {
		t.Fatalf("output:\n%s", out.String())
	}
}

func TestBackupFolderIsExclusivePerRun(t *testing.T) {
	home := t.TempDir()
	needing := plan{harnesses: []harnessPlan{{skills: skills.Plan{Changed: []string{"voice"}}}}}
	first, err := reserveBackup(home, "20260903-100405", needing)
	if err != nil {
		t.Fatal(err)
	}
	second, err := reserveBackup(home, "20260903-100405", needing)
	if err != nil {
		t.Fatal(err)
	}
	if first == second || !strings.HasSuffix(second, "20260903-100405-2") {
		t.Fatalf("first = %q, second = %q", first, second)
	}
	if !exists(first) || !exists(second) {
		t.Fatal("reserved folders were not created")
	}
	idle, err := reserveBackup(home, "20260903-110000", plan{harnesses: []harnessPlan{{letter: letter.Change{Outcome: letter.Same}}}})
	if err != nil {
		t.Fatal(err)
	}
	if exists(idle) {
		t.Fatalf("%q was created with nothing to back up", idle)
	}
}

func TestConfigSaveLeavesOnlyTheConfigFile(t *testing.T) {
	home := t.TempDir()
	if err := saveConfig(home, Config{Harnesses: []string{"pi"}}); err != nil {
		t.Fatal(err)
	}
	if err := saveConfig(home, Config{Harnesses: []string{"claude"}}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(home, ".jstack"))
	if err != nil || len(entries) != 1 || entries[0].Name() != "config.json" {
		t.Fatalf("entries = %v, %v", entries, err)
	}
	if got := read(t, configPath(home)); !strings.Contains(got, `"claude"`) || strings.Contains(got, "pi") {
		t.Fatalf("config = %q", got)
	}
}
