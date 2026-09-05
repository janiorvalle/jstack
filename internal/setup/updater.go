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

// location is where a present tool's binary was found and who owns it.
type location struct {
	path  string
	owner owner
}

// locator answers where a binary is and what the Homebrew prefix is, each
// through the shell setup runs its lines in. The prefix is read once per
// run, the first time a tool's path has to be compared with it.
type locator struct {
	ctx        context.Context
	opts       Options
	prefix     string
	prefixRead bool
}

// locate finds a present tool's binary and who owns it. npm first, by the
// install line: node itself may live under Homebrew, and npm's copy is still
// npm's to update. Then by path: under the Homebrew prefix, in the
// installer's folder, or anywhere else.
func (l *locator) locate(tool tools.Tool) location {
	if tool.NpmInstalled() {
		return location{path: l.pathOf(tool), owner: byNpm}
	}
	path := l.pathOf(tool)
	switch {
	case path == "":
		return location{owner: byInstaller}
	case l.underHomebrew(path):
		return location{path: path, owner: byHomebrew}
	case within(installFolder(l.opts), path):
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

func (l *locator) underHomebrew(path string) bool {
	if runtime.GOOS == "windows" {
		return false
	}
	if !l.prefixRead {
		l.prefix = l.firstLine("brew --prefix")
		l.prefixRead = true
	}
	return l.prefix != "" && within(l.prefix, path)
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
		return "brew upgrade " + status.formula()
	case bySomethingElse:
		return ""
	}
	return status.tool.Command
}

// formula is the Homebrew formula for the tool: the Formula line, or the
// binary's name, which is how formulas are named unless the line says
// otherwise.
func (status toolStatus) formula() string {
	if status.tool.Formula != "" {
		return status.tool.Formula
	}
	return status.tool.Binary
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
