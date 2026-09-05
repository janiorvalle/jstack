package setup

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
)

// ownedFixture is a tools.md whose Check lines name one binary each, the
// way the real file does, so setup can ask the shell where each one is.
func ownedFixture() fstest.MapFS {
	files := fixture()
	files["tools.md"] = &fstest.MapFile{Data: []byte("# Tools\n\n" +
		"## git\n\n- Check: `check-git`\n\n" +
		"## TruffleHog\n\n- Repo: https://github.com/x/trufflehog\n- Check: `command -v trufflehog`\n- Check (windows): `Get-Command trufflehog`\n- Version: `trufflehog --version`\n- Install: `curl trufflehog | sh`\n\n" +
		"## bgr\n\n- Repo: https://github.com/x/bgr\n- Check: `command -v bgr`\n- Check (windows): `Get-Command bgr`\n- Version: `bgr --version`\n- Install: `curl bgr | sh`\n\n" +
		"## browser\n\n- Repo: https://github.com/x/browser\n- Check: `command -v browser`\n- Check (windows): `Get-Command browser`\n- Version: `browser --version`\n- Install: `" + pinnedInstall + "`\n")}
	return files
}

// linked puts a file at target and a symlink to it at link, the way brew
// and npm link what they install into a bin folder.
func linked(t *testing.T, link, target string) {
	t.Helper()
	write(t, target, "#!/bin/sh\n")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
}

// owned is a machine with every tool outdated and each binary where its
// owner put it: TruffleHog linked from Homebrew's bin into its Cellar, bgr
// in ~/.local/bin, browser linked from node's bin into its global
// node_modules. brew and node live under home so the links are real.
func owned(t *testing.T, home string) (*fakeShell, Options) {
	t.Helper()
	brew, node := filepath.Join(home, "brew"), filepath.Join(home, "node")
	linked(t, filepath.Join(brew, "bin", "trufflehog"), filepath.Join(brew, "Cellar", "trufflehog", "3.97.0", "bin", "trufflehog"))
	linked(t, filepath.Join(node, "bin", "browser"), filepath.Join(node, "lib", "node_modules", "browser", "bin", "browser"))
	write(t, filepath.Join(home, ".local", "bin", "bgr"), "#!/bin/sh\n")
	shell := &fakeShell{
		present: map[string]bool{"check-git": true},
		versions: map[string]string{
			"trufflehog --version":  "trufflehog 3.97.0",
			"bgr --version":         "bgr 1.6.0",
			"browser --version":     "browser 0.35.0",
			"brew --prefix":         brew,
			"npm prefix -g":         node,
			"command -v trufflehog": filepath.Join(brew, "bin", "trufflehog"),
			"command -v bgr":        filepath.Join(home, ".local", "bin", "bgr"),
			"command -v browser":    filepath.Join(node, "bin", "browser"),
		},
		latest: map[string]string{"TruffleHog": "v3.97.4", "bgr": "v1.7.0"},
	}
	opts, _ := options(t, home, shell, "")
	opts.Files = ownedFixture()
	return shell, opts
}

func resolved(t *testing.T, opts Options) map[string]toolStatus {
	t.Helper()
	session, err := Start(opts)
	if err != nil {
		t.Fatal(err)
	}
	byTitle := map[string]toolStatus{}
	for _, status := range session.toolStatuses(context.Background()) {
		byTitle[status.tool.Title] = status
	}
	return byTitle
}

func skipOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("Homebrew, ~/.local/bin, and symlinks are not how Windows keeps tools")
	}
}

func TestUpdateGoesThroughWhoeverOwnsTheBinary(t *testing.T) {
	skipOnWindows(t)
	home := homeWithClaude(t)
	_, opts := owned(t, home)
	statuses := resolved(t, opts)
	for title, want := range map[string]string{
		"TruffleHog": "brew upgrade trufflehog",
		"bgr":        "curl bgr | sh",
		"browser":    pinnedInstall,
	} {
		if got := statuses[title].line(); got != want {
			t.Fatalf("%s update = %q, want %q (owner %d at %q)", title, got, want, statuses[title].owner, statuses[title].path)
		}
	}
	if statuses["TruffleHog"].owner != byHomebrew || statuses["bgr"].owner != byInstaller || statuses["browser"].owner != byNpm {
		t.Fatalf("owners: %+v", statuses)
	}
	if got := toolState(statuses["TruffleHog"]); got != "outdated TruffleHog 3.97.0, latest 3.97.4. update: brew upgrade trufflehog" {
		t.Fatalf("plan line = %q", got)
	}
	if got := toolOffer(statuses["TruffleHog"]); got != "update TruffleHog 3.97.0 to 3.97.4: brew upgrade trufflehog" {
		t.Fatalf("offer = %q", got)
	}
}

func TestTheFormulaIsReadOffTheCellarPath(t *testing.T) {
	skipOnWindows(t)
	home := homeWithClaude(t)
	shell, opts := owned(t, home)
	brew := filepath.Join(home, "brew")
	linked(t, filepath.Join(brew, "bin", "bgr"), filepath.Join(brew, "Cellar", "better-git-review", "1.6.0", "bin", "bgr"))
	shell.versions["command -v bgr"] = filepath.Join(brew, "bin", "bgr")
	if got := resolved(t, opts)["bgr"].line(); got != "brew upgrade better-git-review" {
		t.Fatalf("bgr update = %q", got)
	}
}

func TestAFileInHomebrewsBinThatIsNotBrewsLinkIsSomebodyElses(t *testing.T) {
	skipOnWindows(t)
	home := homeWithClaude(t)
	shell, opts := owned(t, home)
	stray := filepath.Join(home, "brew", "bin", "trufflehog")
	if err := os.Remove(stray); err != nil {
		t.Fatal(err)
	}
	write(t, stray, "#!/bin/sh\n")
	shell.versions["bgr --version"] = "bgr 1.7.0"
	shell.versions["browser --version"] = "browser 0.36.0"
	opts.UpdateTools = true
	opts.Yes = true
	status := resolved(t, opts)["TruffleHog"]
	if status.actionable() || status.line() != "" || status.owner != bySomethingElse {
		t.Fatalf("status = %+v", status)
	}
	if got := toolState(status); got != "outdated TruffleHog 3.97.0, latest 3.97.4, at "+stray+", which setup didn't put there; update it the way it was installed" {
		t.Fatalf("plan line = %q", got)
	}
	shell.commands = nil
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if commands := strings.Join(shell.commands, ";"); strings.Contains(commands, "brew upgrade") || strings.Contains(commands, "curl trufflehog") {
		t.Fatalf("--update-tools ran a line over a binary nobody setup knows owns: %v", shell.commands)
	}
}

func TestAnNpmLineBinaryOutsideNodesModulesIsSomebodyElses(t *testing.T) {
	skipOnWindows(t)
	home := homeWithClaude(t)
	shell, opts := owned(t, home)
	elsewhere := filepath.Join(home, "pnpm", "browser")
	write(t, elsewhere, "#!/bin/sh\n")
	shell.versions["command -v browser"] = elsewhere
	status := resolved(t, opts)["browser"]
	if status.actionable() || status.owner != bySomethingElse || status.path != elsewhere {
		t.Fatalf("status = %+v", status)
	}
}

func TestALinkInTheInstallersFolderThatLeadsElsewhereIsSomebodyElses(t *testing.T) {
	skipOnWindows(t)
	home := homeWithClaude(t)
	shell, opts := owned(t, home)
	link := filepath.Join(home, ".local", "bin", "bgr")
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	linked(t, link, filepath.Join(home, ".asdf", "installs", "bgr", "1.6.0", "bin", "bgr"))
	shell.versions["command -v bgr"] = link
	status := resolved(t, opts)["bgr"]
	if status.actionable() || status.owner != bySomethingElse {
		t.Fatalf("status = %+v", status)
	}
}

func TestAFileInsideALinkedInstallersFolderIsTheInstallers(t *testing.T) {
	skipOnWindows(t)
	home := homeWithClaude(t)
	shell, opts := owned(t, home)
	elsewhere := filepath.Join(home, "elsewhere", "bin")
	if err := os.MkdirAll(filepath.Dir(elsewhere), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(home, ".local", "bin"), elsewhere); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, filepath.Join(home, ".local", "bin")); err != nil {
		t.Fatal(err)
	}
	shell.versions["command -v bgr"] = filepath.Join(home, ".local", "bin", "bgr")
	if status := resolved(t, opts)["bgr"]; status.owner != byInstaller || status.line() != "curl bgr | sh" {
		t.Fatalf("status = %+v", status)
	}
}

func TestPrefixesAreReadOnceAndOnlyForOutdatedTools(t *testing.T) {
	skipOnWindows(t)
	home := homeWithClaude(t)
	shell, opts := owned(t, home)
	shell.versions["bgr --version"] = "bgr 1.7.0"
	resolved(t, opts)
	commands := strings.Join(shell.commands, ";")
	if strings.Count(commands, "brew --prefix") != 1 || strings.Count(commands, "npm prefix -g") != 1 || strings.Count(commands, "command -v bgr") != 1 || strings.Count(commands, "command -v trufflehog") != 2 {
		t.Fatalf("commands = %v", shell.commands)
	}
}

func TestBrewThatStaysBehindTheReleaseSaysSo(t *testing.T) {
	skipOnWindows(t)
	home := homeWithClaude(t)
	shell, opts := owned(t, home)
	opts.UpdateTools = true
	opts.Yes = true
	err := Run(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "TruffleHog: `brew upgrade trufflehog` ran, but `trufflehog --version` still prints 3.97.0 and the latest is 3.97.4; Homebrew's formula is behind the release, rerun jstack setup once `brew upgrade` has it") {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(strings.Join(shell.commands, ";"), "brew upgrade trufflehog") {
		t.Fatalf("commands = %v", shell.commands)
	}
}

func TestWithinIsTheFolderOrInsideIt(t *testing.T) {
	for path, want := range map[string]bool{
		"/opt/homebrew/bin/trufflehog": true,
		"/opt/homebrew":                true,
		"/opt/homebrewer/bin/x":        false,
		"/usr/local/bin/x":             false,
	} {
		if got := within("/opt/homebrew", path); got != want {
			t.Fatalf("within(%q) = %v, want %v", path, got, want)
		}
	}
	if within("", "/anything") {
		t.Fatal("an empty folder holds something")
	}
}

func TestResolveLineIsTheShellsOwn(t *testing.T) {
	if got := resolveLine("darwin", "roast"); got != "command -v roast" {
		t.Fatalf("darwin = %q", got)
	}
	if got := resolveLine("windows", "roast"); got != "(Get-Command roast).Source" {
		t.Fatalf("windows = %q", got)
	}
}
