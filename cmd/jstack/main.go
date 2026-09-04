// Command jstack installs the skills, the letter, and the tools into the coding
// agents on this machine, and keeps itself current from GitHub releases.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/janiorvalle/jstack"
	"github.com/janiorvalle/jstack/internal/prompt"
	"github.com/janiorvalle/jstack/internal/setup"
	"github.com/janiorvalle/jstack/internal/tools"
	"github.com/janiorvalle/jstack/internal/upgrade"
)

var version = "dev"

const usage = `jstack puts the skills, the letter, and the tools into the coding agents on this machine.

  jstack setup [--harness claude,codex|all] [--install-tools] [--update-tools] [--keep-instructions] [--yes]
  jstack upgrade
  jstack version

setup prints the plan first. With a terminal it asks which harnesses and which tools, then applies.
Without one it changes nothing unless --yes is passed. Picks are saved in ~/.jstack/config.json.
Each tool is missing, outdated, or current; the latest versions come from GitHub and npm.
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

type dependencies struct {
	setup       func(context.Context, setup.Options) error
	upgrade     func(context.Context, string, io.Writer) error
	home        func() (string, error)
	interactive func() bool
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return runWith(args, stdin, stdout, stderr, dependencies{
		setup:       setup.Run,
		upgrade:     upgrade.Run,
		home:        os.UserHomeDir,
		interactive: stdinIsTerminal,
	})
}

func runWith(args []string, stdin io.Reader, stdout, stderr io.Writer, deps dependencies) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	// setup keeps the default signal handling: Ctrl-C at a question ends the
	// process before anything is written. upgrade turns signals into a
	// cancelled context so a download stops cleanly.
	switch args[0] {
	case "setup":
		return runSetup(context.Background(), args[1:], stdin, stdout, stderr, deps)
	case "upgrade":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return runUpgrade(ctx, args[1:], stdout, stderr, deps)
	case "version", "--version", "-v":
		fmt.Fprintln(stdout, version)
		return 0
	case "help", "--help", "-h":
		fmt.Fprint(stdout, usage)
		return 0
	case "__upgrade-cleanup":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return runUpgradeCleanup(ctx, args[1:], stderr)
	}
	fmt.Fprintf(stderr, "jstack: [JSTACK-CLI-COMMAND] unknown command %q; use `jstack setup`, `jstack upgrade`, or `jstack version`\n", args[0])
	return 2
}

func runSetup(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer, deps dependencies) int {
	flags := flag.NewFlagSet("jstack setup", flag.ContinueOnError)
	flags.SetOutput(stderr)
	harnesses := flags.String("harness", "", "harnesses to install into: a comma-separated list of claude, codex, opencode, cursor, pi, or all")
	installTools := flags.Bool("install-tools", false, "install the missing tools without asking")
	updateTools := flags.Bool("update-tools", false, "update the outdated tools without asking")
	keepInstructions := flags.Bool("keep-instructions", false, "append the letter to an instructions file that has other content instead of replacing it")
	yes := flags.Bool("yes", false, "apply without asking; without a terminal nothing changes unless this is set")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "jstack: [JSTACK-CLI-ARGS] setup takes flags only, got %q; example: jstack setup --harness claude,codex --yes\n", flags.Arg(0))
		return 2
	}
	home, err := deps.home()
	if err != nil {
		fmt.Fprintf(stderr, "jstack: [JSTACK-HOME] cannot find the home directory: %v; set HOME and rerun\n", err)
		return 1
	}
	err = deps.setup(ctx, setup.Options{
		Files:            jstack.Files,
		Home:             home,
		Getenv:           os.Getenv,
		Harness:          *harnesses,
		InstallTools:     *installTools,
		UpdateTools:      *updateTools,
		KeepInstructions: *keepInstructions,
		Yes:              *yes,
		Interactive:      deps.interactive(),
		Stdin:            stdin,
		Stdout:           stdout,
		Shell:            runShell,
		Latest:           tools.Lookup{Client: &http.Client{Timeout: 5 * time.Second}}.Latest,
		Now:              time.Now,
	})
	if errors.Is(err, prompt.ErrQuit) {
		fmt.Fprintln(stdout, "\nQuit. Nothing changed.")
		return 0
	}
	if err != nil {
		fmt.Fprintf(stderr, "jstack: %v\n", err)
		return 1
	}
	return 0
}

func runUpgrade(ctx context.Context, args []string, stdout, stderr io.Writer, deps dependencies) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "jstack: [JSTACK-UPGRADE-ARGS] upgrade takes no arguments; use `jstack upgrade`")
		return 2
	}
	if err := deps.upgrade(ctx, version, stdout); err != nil {
		fmt.Fprintf(stderr, "jstack: %v\n", err)
		return 1
	}
	return 0
}

func runUpgradeCleanup(ctx context.Context, args []string, stderr io.Writer) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, "jstack: [JSTACK-UPGRADE-CLEANUP] internal cleanup expected a backup path and parent process ID; rerun `jstack upgrade`")
		return 2
	}
	parentProcessID, err := strconv.Atoi(args[1])
	if err != nil {
		fmt.Fprintf(stderr, "jstack: [JSTACK-UPGRADE-CLEANUP] parent process ID %q is invalid; expected a positive integer; rerun `jstack upgrade`\n", args[1])
		return 2
	}
	if err := upgrade.CleanupPreviousExecutable(ctx, args[0], parentProcessID); err != nil {
		fmt.Fprintf(stderr, "jstack: %v\n", err)
		return 1
	}
	return 0
}

// runShell runs one tools.md line in the shell the OS ships with.
func runShell(ctx context.Context, command string, output io.Writer) error {
	arguments := shellArguments(runtime.GOOS, command)
	process := exec.CommandContext(ctx, arguments[0], arguments[1:]...)
	process.Stdin = os.Stdin
	process.Stdout = output
	process.Stderr = output
	return process.Run()
}

// windowsRefreshPath adds the user PATH from the registry to the one this
// process inherited. A Windows installer puts its folder on the user PATH,
// which only new terminals see, so without this the check that follows an
// install would not find the tool it just installed.
const windowsRefreshPath = "$env:Path += ';' + [Environment]::GetEnvironmentVariable('Path', 'User')"

// posixInstallFolderOnPath puts ~/.local/bin first on PATH when it is not
// there yet. The jstack installer and every curl installer in tools.md put
// their tool in that folder, and a fresh machine has it off PATH until the
// shell profile says otherwise, so without this the check that follows an
// install would not find the tool it just installed. A PATH that already has
// the folder is left in its order.
const posixInstallFolderOnPath = `case ":$PATH:" in *":$HOME/.local/bin:"*) ;; *) PATH="$HOME/.local/bin:$PATH" ;; esac`

// shellArguments is sh on macOS and Linux and Windows PowerShell on Windows,
// the two shells the lines in tools.md are written for. The installer and
// the lines pick the same way, so this is the one place the OS decides, and
// the one place the folder a fresh install lands in is put on PATH.
func shellArguments(operatingSystem, command string) []string {
	if operatingSystem == "windows" {
		return []string{"powershell", "-NoProfile", "-Command", windowsRefreshPath + "; " + command}
	}
	return []string{"sh", "-c", posixInstallFolderOnPath + "; " + command}
}

// stdinIsTerminal is true for a real terminal. /dev/null is a character
// device too, so a run with stdin redirected from it counts as no terminal.
func stdinIsTerminal() bool {
	info, err := os.Stdin.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	null, err := os.Stat(os.DevNull)
	return err != nil || !os.SameFile(info, null)
}
