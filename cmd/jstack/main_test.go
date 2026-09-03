package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/janiorvalle/jstack/internal/prompt"
	"github.com/janiorvalle/jstack/internal/setup"
)

func fakeDependencies(captured *setup.Options, upgraded *bool, err error) dependencies {
	return dependencies{
		setup: func(_ context.Context, opts setup.Options) error {
			if captured != nil {
				*captured = opts
			}
			return err
		},
		upgrade: func(context.Context, string, io.Writer) error {
			*upgraded = true
			return err
		},
		home:        func() (string, error) { return "/home/test", nil },
		interactive: func() bool { return true },
		lookPath:    func(name string) (string, error) { return "/bin/" + name, nil },
	}
}

func TestSetupNeedsAPosixShell(t *testing.T) {
	upgraded := false
	deps := fakeDependencies(nil, &upgraded, nil)
	deps.lookPath = func(string) (string, error) { return "", errors.New("not found") }
	var stdout, stderr bytes.Buffer
	code := runWith([]string{"setup"}, strings.NewReader(""), &stdout, &stderr, deps)
	if code != 1 || !strings.Contains(stderr.String(), "JSTACK-SHELL") {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
}

func TestSetupFlagsReachTheRun(t *testing.T) {
	var captured setup.Options
	upgraded := false
	var stdout, stderr bytes.Buffer
	code := runWith([]string{"setup", "--harness", "claude,codex", "--install-tools", "--keep-instructions", "--yes"}, strings.NewReader(""), &stdout, &stderr, fakeDependencies(&captured, &upgraded, nil))
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if captured.Harness != "claude,codex" || !captured.InstallTools || !captured.KeepInstructions || !captured.Yes || !captured.Interactive || captured.Home != "/home/test" || captured.Files == nil || captured.Shell == nil {
		t.Fatalf("options = %+v", captured)
	}
}

func TestSetupQuitIsNotAnError(t *testing.T) {
	upgraded := false
	var stdout, stderr bytes.Buffer
	code := runWith([]string{"setup"}, strings.NewReader(""), &stdout, &stderr, fakeDependencies(nil, &upgraded, prompt.ErrQuit))
	if code != 0 || !strings.Contains(stdout.String(), "Quit. Nothing changed.") {
		t.Fatalf("code = %d, stdout = %q", code, stdout.String())
	}
}

func TestSetupErrorGoesToStderrWithCodeOne(t *testing.T) {
	upgraded := false
	var stdout, stderr bytes.Buffer
	code := runWith([]string{"setup"}, strings.NewReader(""), &stdout, &stderr, fakeDependencies(nil, &upgraded, errors.New("[JSTACK-X] boom")))
	if code != 1 || !strings.Contains(stderr.String(), "jstack: [JSTACK-X] boom") {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
}

func TestUsageErrors(t *testing.T) {
	upgraded := false
	cases := []struct {
		args []string
		want string
	}{
		{nil, "jstack setup"},
		{[]string{"frobnicate"}, "JSTACK-CLI-COMMAND"},
		{[]string{"setup", "extra"}, "JSTACK-CLI-ARGS"},
		{[]string{"setup", "--bogus"}, "flag provided but not defined"},
		{[]string{"upgrade", "now"}, "JSTACK-UPGRADE-ARGS"},
		{[]string{"__upgrade-cleanup", "x"}, "JSTACK-UPGRADE-CLEANUP"},
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
	if code := runWith([]string{"--help"}, strings.NewReader(""), &stdout, io.Discard, fakeDependencies(nil, &upgraded, nil)); code != 0 || !strings.Contains(stdout.String(), "jstack upgrade") {
		t.Fatalf("help: code = %d, stdout = %q", code, stdout.String())
	}
	if code := runWith([]string{"upgrade"}, strings.NewReader(""), io.Discard, io.Discard, fakeDependencies(nil, &upgraded, nil)); code != 0 || !upgraded {
		t.Fatalf("upgrade: code = %d, upgraded = %t", code, upgraded)
	}
}
