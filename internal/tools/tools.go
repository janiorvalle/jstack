// Package tools reads tools.md, the list of tools the flow expects on the
// machine, into the lines setup runs and shows.
package tools

import (
	"regexp"
	"strings"
)

// Tool is one section of tools.md that carries a Check line.
type Tool struct {
	Title        string
	Check        string
	Install      string
	Command      string
	SkillInstall string
	SkillFolder  string
}

var (
	checkLine        = regexp.MustCompile("(?m)^- Check: `([^`]+)`")
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

func first(pattern *regexp.Regexp, text string) string {
	match := pattern.FindStringSubmatch(text)
	if match == nil {
		return ""
	}
	return match[1]
}
