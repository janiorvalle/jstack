// Package tools reads tools.md, the list of tools the flow expects on the
// machine, into the lines setup runs and shows.
package tools

import (
	"regexp"
	"strings"

	"golang.org/x/mod/semver"
)

// Tool is one section of tools.md that carries a Check line. Version is the
// command that prints the installed version and Repo is the tool's GitHub
// page; both are empty for tools setup never updates, such as git.
type Tool struct {
	Title        string
	Check        string
	Version      string
	Repo         string
	Install      string
	Command      string
	SkillInstall string
	SkillFolder  string
}

var (
	checkLine        = regexp.MustCompile("(?m)^- Check: `([^`]+)`")
	versionLine      = regexp.MustCompile("(?m)^- Version: `([^`]+)`")
	repoLine         = regexp.MustCompile(`(?m)^- Repo: (\S+)$`)
	versionToken     = regexp.MustCompile(`v?\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?`)
	installLine      = regexp.MustCompile(`(?m)^- Install: (.+)$`)
	skillInstallLine = regexp.MustCompile("(?m)^- Skill install: `([^`]+)`")
	skillFolderLine  = regexp.MustCompile("(?m)^- Skill folder: `([^`]+)`")
	backtickSpan     = regexp.MustCompile("`([^`]+)`")
)

// Parse returns the tools in tools.md order. Sections without a Check line
// are prose and are skipped.
func Parse(markdown string) []Tool {
	var parsed []Tool
	for _, section := range regexp.MustCompile(`(?m)^## `).Split(markdown, -1)[1:] {
		title, _, _ := strings.Cut(section, "\n")
		check := first(checkLine, section)
		if check == "" {
			continue
		}
		install := "see tools.md"
		command := ""
		if match := installLine.FindStringSubmatch(section); match != nil {
			install = strings.ReplaceAll(strings.TrimSpace(match[1]), "`", "")
			command = commandFrom(match[1])
		}
		parsed = append(parsed, Tool{
			Title:        strings.TrimSpace(title),
			Check:        check,
			Version:      first(versionLine, section),
			Repo:         first(repoLine, section),
			Install:      install,
			Command:      command,
			SkillInstall: first(skillInstallLine, section),
			SkillFolder:  first(skillFolderLine, section),
		})
	}
	return parsed
}

// commandFrom joins the backtick spans of an Install line into one shell
// line, so "`brew install git gh`, then `gh auth login`" runs both steps.
func commandFrom(line string) string {
	var steps []string
	for _, match := range backtickSpan.FindAllStringSubmatch(line, -1) {
		steps = append(steps, match[1])
	}
	return strings.Join(steps, " && ")
}

// ParseVersion picks the version out of what a tool's Version command
// printed, so "bgr 1.7.0" and "tokenomnom version 0.6.6" both work, and
// returns it with the leading v that semver expects. Output with no version
// in it gives "".
func ParseVersion(output string) string {
	found := versionToken.FindString(output)
	if found == "" {
		return ""
	}
	if !strings.HasPrefix(found, "v") {
		found = "v" + found
	}
	if !semver.IsValid(found) {
		return ""
	}
	return found
}

// Outdated reports whether installed is behind latest. When either side is
// unknown the answer is no: setup only offers an update it can name.
func Outdated(installed, latest string) bool {
	return installed != "" && latest != "" && semver.Compare(installed, latest) < 0
}

// Display shows a version the way the tools print it, without the leading v.
func Display(version string) string {
	return strings.TrimPrefix(version, "v")
}

func first(pattern *regexp.Regexp, text string) string {
	match := pattern.FindStringSubmatch(text)
	if match == nil {
		return ""
	}
	return match[1]
}
