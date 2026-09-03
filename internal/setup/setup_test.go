package setup

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/janiorvalle/jstack/internal/letter"
)

const toolsFixture = "# Tools\n\n" +
	"## git\n\n- Check: `check-git`\n- Install: `brew install git`\n\n" +
	"## roast\n\n- Check: `check-roast`\n- Install: `curl roast | sh`\n- Skill install: `roast install-skill`\n- Skill folder: `roast`\n\n" +
	"## prose only\n\nno check line\n"

func fixture() fstest.MapFS {
	return fstest.MapFS{
		"skills/voice/SKILL.md": {Data: []byte("voice v2\n")},
		"skills/how/SKILL.md":   {Data: []byte("how\n")},
		"AGENTS.md":             {Data: []byte("# the letter\n")},
		"tools.md":              {Data: []byte(toolsFixture)},
		"vendor.json":           {Data: []byte(`{"skills":[{"name":"how"}]}`)},
	}
}

type fakeShell struct {
	present  map[string]bool
	commands []string
}

func (f *fakeShell) run(_ context.Context, command string, _ io.Writer) error {
	f.commands = append(f.commands, command)
	if strings.HasPrefix(command, "check-") && !f.present[command] {
		return errors.New("exit status 1")
	}
	return nil
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
		Stdin:  strings.NewReader(stdin),
		Stdout: &out,
		Shell:  shell.run,
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
	shell := &fakeShell{present: map[string]bool{"check-git": true}}
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
	shell := &fakeShell{present: map[string]bool{"check-git": true, "check-roast": true}}
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
	if strings.Join(shell.commands, ";") != "check-git;check-roast;roast install-skill" {
		t.Fatalf("commands = %v", shell.commands)
	}
	for _, expected := range []string{
		"skills   1 installed, 1 updated in ~/.claude/skills",
		"backup   ~/.jstack/backup/20260903-100405/claude/skills",
		"letter   replaced ~/.claude/CLAUDE.md, old file backed up to ~/.jstack/backup/20260903-100405/claude/CLAUDE.md",
		"ok roast, skill installed via roast install-skill",
		"restart the harness so the skills load",
	} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("output missing %q:\n%s", expected, out.String())
		}
	}
}

func TestSecondRunUsesSavedPicksAndReportsUpToDate(t *testing.T) {
	home := homeWithClaude(t)
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	shell := &fakeShell{present: map[string]bool{"check-git": true, "check-roast": true}}
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
	for _, expected := range []string{"Apply to Claude Code, Codex? [Y/n]", "skills   up to date in ~/.claude/skills", "letter   up to date in ~/.claude/CLAUDE.md", "ok roast, skill present"} {
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
	shell := &fakeShell{present: map[string]bool{"check-git": true}}
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
		"installed roast",
		"ok roast, skill installed via roast install-skill",
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
	if strings.Join(shell.commands, ";") != "check-git;check-roast;curl roast | sh;roast install-skill" {
		t.Fatalf("commands = %v", shell.commands)
	}
}

func TestTerminalNoToToolKeepsItMissing(t *testing.T) {
	home := homeWithClaude(t)
	shell := &fakeShell{present: map[string]bool{"check-git": true}}
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
	shell := &fakeShell{present: map[string]bool{"check-git": true, "check-roast": true}}
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
	shell := &fakeShell{present: map[string]bool{"check-git": true}}
	opts, _ := options(t, home, shell, "")
	opts.Yes = true
	opts.InstallTools = true
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if strings.Join(shell.commands, ";") != "check-git;check-roast;curl roast | sh;roast install-skill" {
		t.Fatalf("commands = %v", shell.commands)
	}
}

func TestKeepInstructionsAppendsTheLetter(t *testing.T) {
	home := homeWithClaude(t)
	shell := &fakeShell{present: map[string]bool{"check-git": true, "check-roast": true}}
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
	shell := &fakeShell{present: map[string]bool{"check-git": true, "check-roast": true}}
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
