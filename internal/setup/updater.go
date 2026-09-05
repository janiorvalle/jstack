package setup

import (
	"bytes"
	"context"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/janiorvalle/jstack/internal/tools"
)

// owner is who put a tool's binary where it is. The tools.md install line
// only updates what it installed, so the owner decides the update offer: a
// binary Homebrew put under its prefix gets brew's line, and one something
// else put somewhere setup doesn't know gets no offer, since the install
// line would drop a second copy that loses to the first on PATH.
type owner int

const (
	// byInstaller is the tools.md install line's own folder, ~/.local/bin on
	// macOS and Linux and %LOCALAPPDATA%\Programs on Windows. Also what a
	// tool setup can't locate is taken as: the check passed, so the tool is
	// there, and nothing says it's anywhere else.
	byInstaller owner = iota
	byNpm
	byHomebrew
	bySomethingElse
)

// location is where a present tool's binary was found, who owns it, and
// for Homebrew the formula, read off the Cellar path.
type location struct {
	path    string
	owner   owner
	formula string
}

// locator answers where a binary is, where Homebrew and npm keep what they
// install, each through the shell setup runs its lines in. The prefixes
// are read once per run, the first time a tool's path has to be compared
// with them.
type locator struct {
	ctx      context.Context
	opts     Options
	prefixes map[string]string
}

// locate finds a present tool's binary and who owns it, by where the
// binary really is once links are followed: an npm install line's binary
// under node's global node_modules, or a link into Homebrew's Cellar, or
// a file in the installer's folder. A binary in Homebrew's bin that isn't
// a link into the Cellar is not Homebrew's, whatever put it there, and
// brew upgrade would leave it standing.
func (l *locator) locate(tool tools.Tool) location {
	path := l.pathOf(tool)
	if path == "" {
		return location{owner: byInstaller}
	}
	real := path
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		real = resolved
	}
	if tool.NpmInstalled() {
		if within(l.npmModules(), real) {
			return location{path: path, owner: byNpm}
		}
		return location{path: path, owner: bySomethingElse}
	}
	if formula := l.homebrewFormula(real); formula != "" {
		return location{path: path, owner: byHomebrew, formula: formula}
	}
	if within(installFolder(l.opts), path) {
		return location{path: path, owner: byInstaller}
	}
	return location{path: path, owner: bySomethingElse}
}

// pathOf is what the shell resolves the tool's binary to, "" for a tool
// whose Check line names no single binary or that the shell can't find.
func (l *locator) pathOf(tool tools.Tool) string {
	if tool.Binary == "" {
		return ""
	}
	return l.firstLine(resolveLine(runtime.GOOS, tool.Binary))
}

// resolveLine prints the path of a binary on PATH, in the shell of the OS.
func resolveLine(operatingSystem, binary string) string {
	if operatingSystem == "windows" {
		return "(Get-Command " + binary + ").Source"
	}
	return "command -v " + binary
}

// homebrewFormula is the formula a binary belongs to, read off its real
// path under the Cellar, <prefix>/Cellar/<formula>/<version>/..., or ""
// when it isn't there. Windows has no Homebrew.
func (l *locator) homebrewFormula(real string) string {
	if runtime.GOOS == "windows" {
		return ""
	}
	prefix := l.prefix("brew --prefix")
	if prefix == "" {
		return ""
	}
	cellar := filepath.Join(prefix, "Cellar")
	if resolved, err := filepath.EvalSymlinks(cellar); err == nil {
		cellar = resolved
	}
	if !within(cellar, real) {
		return ""
	}
	relative, err := filepath.Rel(cellar, real)
	if err != nil {
		return ""
	}
	formula, _, _ := strings.Cut(filepath.ToSlash(relative), "/")
	return formula
}

// npmModules is where npm keeps the global packages: lib/node_modules under
// the global prefix on macOS and Linux, the prefix itself on Windows, where
// the .cmd shims sit beside node_modules.
func (l *locator) npmModules() string {
	prefix := l.prefix("npm prefix -g")
	if prefix == "" {
		return ""
	}
	if runtime.GOOS == "windows" {
		return prefix
	}
	modules := filepath.Join(prefix, "lib", "node_modules")
	if resolved, err := filepath.EvalSymlinks(modules); err == nil {
		modules = resolved
	}
	return modules
}

// prefix runs a command that prints a folder, once per run.
func (l *locator) prefix(command string) string {
	if l.prefixes == nil {
		l.prefixes = map[string]string{}
	}
	if answer, done := l.prefixes[command]; done {
		return answer
	}
	l.prefixes[command] = l.firstLine(command)
	return l.prefixes[command]
}

func (l *locator) firstLine(command string) string {
	var output bytes.Buffer
	if err := l.opts.Shell(l.ctx, command, &output); err != nil {
		return ""
	}
	line, _, _ := strings.Cut(strings.TrimSpace(output.String()), "\n")
	return strings.TrimSpace(line)
}

// installFolder is where the tools.md install lines put their binaries.
func installFolder(opts Options) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(opts.Getenv("LOCALAPPDATA"), "Programs")
	}
	return filepath.Join(opts.Home, ".local", "bin")
}

// within reports whether path is folder or inside it.
func within(folder, path string) bool {
	if folder == "" {
		return false
	}
	relative, err := filepath.Rel(filepath.Clean(folder), filepath.Clean(path))
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

// update is the line that brings an outdated tool to the latest where it
// is, "" when nobody setup knows put it there.
func (status toolStatus) update() string {
	switch status.owner {
	case byHomebrew:
		return "brew upgrade " + status.formula
	case bySomethingElse:
		return ""
	}
	return status.tool.Command
}

// updateShown is the update line as the plan and the report show it: the
// tools.md text for the install line, which may carry words for a person,
// or the brew line as it runs.
func (status toolStatus) updateShown() string {
	if status.owner == byHomebrew {
		return status.update()
	}
	return status.tool.Install
}
