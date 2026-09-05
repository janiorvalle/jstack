package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/janiorvalle/squirrel/internal/setup"
	"github.com/janiorvalle/squirrel/internal/tui"
)

// fakeDependencies captures the options either setup path gets and records
// which one ran, guided for the screens or setup for the flags.
func fakeDependencies(captured *setup.Options, upgraded *bool, err error) dependencies {
	return fakeDependenciesRan(captured, upgraded, err, new(string))
}

func fakeDependenciesRan(captured *setup.Options, upgraded *bool, err error, ran *string) dependencies {
	capture := func(path string) func(context.Context, setup.Options) error {
		return func(_ context.Context, opts setup.Options) error {
			*ran = path
			if captured != nil {
				*captured = opts
			}
			return err
		}
	}
	return dependencies{
		setup:  capture("setup"),
		guided: capture("guided"),
		upgrade: func(context.Context, string, io.Writer) error {
			*upgraded = true
			return err
		},
		home:        func() (string, error) { return "/home/test", nil },
		interactive: func() bool { return true },
	}
}

func TestSetupFlagsReachTheRun(t *testing.T) {
	var captured setup.Options
	upgraded := false
	var stdout, stderr bytes.Buffer
	code := runWith([]string{"setup", "--harness", "claude,codex", "--install-tools", "--update-tools", "--keep-instructions", "--yes"}, strings.NewReader(""), &stdout, &stderr, fakeDependencies(&captured, &upgraded, nil))
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if captured.Harness != "claude,codex" || !captured.InstallTools || !captured.UpdateTools || !captured.KeepInstructions || !captured.Yes || captured.Home != "/home/test" || captured.Files == nil || captured.Shell == nil || captured.Stdin == nil {
		t.Fatalf("options = %+v", captured)
	}
	t.Setenv("CODEX_HOME", "/work/codex")
	if captured.Getenv == nil || captured.Getenv("CODEX_HOME") != "/work/codex" {
		t.Fatal("Getenv does not read the process environment")
	}
}

func TestSkillRepoFlagsReachTheRunAndRepeat(t *testing.T) {
	var captured setup.Options
	upgraded := false
	var stdout, stderr bytes.Buffer
	code := runWith([]string{"setup", "--skill-repo", "me/work-skills", "--skill-repo", "me/more", "--forget-skill-repo", "me/old", "--no-skill-repo", "--override", "voice=me/work-skills", "--override", "how=squirrel", "--repos-dir", "/home/test/code", "--repos-dir", "/home/test/src", "--ask-trackers-again"}, strings.NewReader(""), &stdout, &stderr, fakeDependencies(&captured, &upgraded, nil))
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if strings.Join(captured.SkillRepos, ",") != "me/work-skills,me/more" || strings.Join(captured.ForgetSkillRepos, ",") != "me/old" || !captured.NoSkillRepo || captured.Overrides["voice"] != "me/work-skills" || captured.Overrides["how"] != "squirrel" || strings.Join(captured.ReposDirs, ",") != "/home/test/code,/home/test/src" || !captured.AskTrackersAgain {
		t.Fatalf("options = %+v", captured)
	}
	code = runWith([]string{"setup", "--override", "voice"}, strings.NewReader(""), &stdout, &stderr, fakeDependencies(&captured, &upgraded, nil))
	if code != 2 || !strings.Contains(stderr.String(), "[SQUIRREL-CLI-OVERRIDE] --override needs skill=source, got \"voice\"; example: --override land-pr=janiorvalle/work-skills") {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
}

func TestTerminalGetsTheScreensAndYesOrNoTerminalGetsTheFlags(t *testing.T) {
	upgraded := false
	var stdout, stderr bytes.Buffer
	for _, run := range []struct {
		args        []string
		interactive bool
		want        string
	}{
		{[]string{"setup"}, true, "guided"},
		{[]string{"setup", "--yes"}, true, "setup"},
		{[]string{"setup"}, false, "setup"},
		{[]string{"setup", "--yes"}, false, "setup"},
	} {
		ran := ""
		deps := fakeDependenciesRan(nil, &upgraded, nil, &ran)
		deps.interactive = func() bool { return run.interactive }
		if code := runWith(run.args, strings.NewReader(""), &stdout, &stderr, deps); code != 0 || ran != run.want {
			t.Fatalf("%v with terminal %v: code = %d, ran %q, want %q", run.args, run.interactive, code, ran, run.want)
		}
	}
}

func TestSetupQuitIsNotAnError(t *testing.T) {
	upgraded := false
	var stdout, stderr bytes.Buffer
	code := runWith([]string{"setup"}, strings.NewReader(""), &stdout, &stderr, fakeDependencies(nil, &upgraded, tui.ErrQuit))
	if code != 0 || !strings.Contains(stdout.String(), "Quit. Nothing changed.") {
		t.Fatalf("code = %d, stdout = %q", code, stdout.String())
	}
}

func TestSetupErrorGoesToStderrWithCodeOne(t *testing.T) {
	upgraded := false
	var stdout, stderr bytes.Buffer
	code := runWith([]string{"setup"}, strings.NewReader(""), &stdout, &stderr, fakeDependencies(nil, &upgraded, errors.New("[SQUIRREL-X] boom")))
	if code != 1 || !strings.Contains(stderr.String(), "squirrel: [SQUIRREL-X] boom") {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
}

func TestUsageErrors(t *testing.T) {
	upgraded := false
	cases := []struct {
		args []string
		want string
	}{
		{nil, "squirrel setup"},
		{[]string{"frobnicate"}, "SQUIRREL-CLI-COMMAND"},
		{[]string{"setup", "extra"}, "SQUIRREL-CLI-ARGS"},
		{[]string{"setup", "--bogus"}, "flag provided but not defined"},
		{[]string{"upgrade", "now"}, "SQUIRREL-UPGRADE-ARGS"},
		{[]string{"__upgrade-cleanup", "x"}, "SQUIRREL-UPGRADE-CLEANUP"},
	}
	for _, tc := range cases {
		var stdout, stderr bytes.Buffer
		code := runWith(tc.args, strings.NewReader(""), &stdout, &stderr, fakeDependencies(nil, &upgraded, nil))
		if code != 2 || !strings.Contains(stderr.String(), tc.want) {
			t.Fatalf("args %v: code = %d, stderr = %q", tc.args, code, stderr.String())
		}
	}
	if upgraded {
		t.Fatal("upgrade ran on a usage error")
	}
}

func TestVersionAndHelpAndUpgrade(t *testing.T) {
	upgraded := false
	for _, args := range [][]string{{"version"}, {"--version"}} {
		var stdout bytes.Buffer
		if code := runWith(args, strings.NewReader(""), &stdout, io.Discard, fakeDependencies(nil, &upgraded, nil)); code != 0 || stdout.String() != version+"\n" {
			t.Fatalf("%v: code = %d, stdout = %q", args, code, stdout.String())
		}
	}
	var stdout bytes.Buffer
	if code := runWith([]string{"--help"}, strings.NewReader(""), &stdout, io.Discard, fakeDependencies(nil, &upgraded, nil)); code != 0 || !strings.Contains(stdout.String(), "squirrel upgrade") {
		t.Fatalf("help: code = %d, stdout = %q", code, stdout.String())
	}
	if code := runWith([]string{"upgrade"}, strings.NewReader(""), io.Discard, io.Discard, fakeDependencies(nil, &upgraded, nil)); code != 0 || !upgraded {
		t.Fatalf("upgrade: code = %d, upgraded = %t", code, upgraded)
	}
}

func TestShellIsPickedByOS(t *testing.T) {
	for goos, want := range map[string]string{
		"darwin":  "sh -c " + posixInstallFolderOnPath + "; command -v quest",
		"linux":   "sh -c " + posixInstallFolderOnPath + "; command -v quest",
		"windows": "powershell -NoProfile -Command " + windowsRefreshPath + "; command -v quest",
	} {
		if got := strings.Join(shellArguments(goos, "command -v quest"), " "); got != want {
			t.Errorf("%s: shell = %q, want %q", goos, got, want)
		}
	}
}

// A tool that a curl installer just put in ~/.local/bin is found by the check
// that follows, even when that folder is not on the PATH setup started with.
func TestShellFindsAToolJustInstalledInLocalBin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh only")
	}
	home := t.TempDir()
	folder := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "quest"), []byte("#!/bin/sh\necho 0.1.0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/bin:/bin")
	var output bytes.Buffer
	if err := runShell(context.Background(), "command -v quest && quest --version", &output); err != nil {
		t.Fatalf("check failed: %v, output %q", err, output.String())
	}
	if !strings.Contains(output.String(), folder+"/quest\n0.1.0\n") {
		t.Fatalf("output = %q, want the tool in %s", output.String(), folder)
	}
}

// The folder goes first only when it is missing, so a PATH that already has
// it, wherever it sits, is left alone.
func TestShellKeepsAPathThatHasLocalBin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh only")
	}
	home := t.TempDir()
	path := "/usr/bin:/bin:" + filepath.Join(home, ".local", "bin")
	t.Setenv("HOME", home)
	t.Setenv("PATH", path)
	var output bytes.Buffer
	if err := runShell(context.Background(), "printf '%s' \"$PATH\"", &output); err != nil {
		t.Fatal(err)
	}
	if output.String() != path {
		t.Fatalf("PATH = %q, want %q unchanged", output.String(), path)
	}
}
