package setup

import (
	"context"
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
		"## bgr\n\n- Repo: https://github.com/x/bgr\n- Check: `command -v bgr`\n- Check (windows): `Get-Command bgr`\n- Version: `bgr --version`\n- Install: `curl bgr | sh`\n- Formula: `better-git-review`\n\n" +
		"## browser\n\n- Repo: https://github.com/x/browser\n- Check: `command -v browser`\n- Check (windows): `Get-Command browser`\n- Version: `browser --version`\n- Install: `" + pinnedInstall + "`\n")}
	return files
}

// owned is a machine with every tool outdated and each binary somewhere:
// TruffleHog under Homebrew, bgr in ~/.local/bin, browser from npm under
// Homebrew's node.
func owned(t *testing.T, home string) (*fakeShell, Options) {
	t.Helper()
	shell := &fakeShell{
		present: map[string]bool{"check-git": true, "command -v trufflehog": true, "command -v bgr": true, "command -v browser": true},
		versions: map[string]string{
			"trufflehog --version":  "trufflehog 3.97.0",
			"bgr --version":         "bgr 1.6.0",
			"browser --version":     "browser 0.35.0",
			"brew --prefix":         "/opt/homebrew",
			"command -v trufflehog": "/opt/homebrew/bin/trufflehog",
			"command -v bgr":        filepath.Join(home, ".local", "bin", "bgr"),
			"command -v browser":    "/opt/homebrew/bin/browser",
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

func TestUpdateGoesThroughWhoeverOwnsTheBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Homebrew and ~/.local/bin are not where Windows keeps tools")
	}
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

func TestFormulaLineNamesTheBrewFormula(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Homebrew is not where Windows keeps tools")
	}
	home := homeWithClaude(t)
	shell, opts := owned(t, home)
	shell.versions["command -v bgr"] = "/opt/homebrew/bin/bgr"
	if got := resolved(t, opts)["bgr"].line(); got != "brew upgrade better-git-review" {
		t.Fatalf("bgr update = %q", got)
	}
}

func TestBinarySomewhereElseGetsThePathAndNoOffer(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Homebrew and ~/.local/bin are not where Windows keeps tools")
	}
	home := homeWithClaude(t)
	shell, opts := owned(t, home)
	shell.versions["command -v bgr"] = "/usr/local/bin/bgr"
	shell.versions["trufflehog --version"] = "trufflehog 3.97.4"
	shell.versions["browser --version"] = "browser 0.36.0"
	opts.UpdateTools = true
	opts.Yes = true
	status := resolved(t, opts)["bgr"]
	if status.actionable() || status.line() != "" || status.owner != bySomethingElse {
		t.Fatalf("status = %+v", status)
	}
	if got := toolState(status); got != "outdated bgr 1.6.0, latest 1.7.0, at /usr/local/bin/bgr, which setup didn't put there; update it the way it was installed" {
		t.Fatalf("plan line = %q", got)
	}
	shell.commands = nil
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(shell.commands, ";"), "curl bgr | sh") {
		t.Fatalf("--update-tools ran the install line over a binary setup didn't put there: %v", shell.commands)
	}
}

func TestBrewPrefixIsReadOnceAndOnlyForAnOutdatedTool(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Homebrew is not where Windows keeps tools")
	}
	home := homeWithClaude(t)
	shell, opts := owned(t, home)
	shell.versions["bgr --version"] = "bgr 1.7.0"
	resolved(t, opts)
	commands := strings.Join(shell.commands, ";")
	if strings.Count(commands, "brew --prefix") != 1 || strings.Count(commands, "command -v bgr") != 1 || strings.Count(commands, "command -v trufflehog") != 2 {
		t.Fatalf("commands = %v", shell.commands)
	}
}

func TestBrewThatStaysBehindTheReleaseSaysSo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Homebrew is not where Windows keeps tools")
	}
	home := homeWithClaude(t)
	shell, opts := owned(t, home)
	shell.present["check-bgr"] = true
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
