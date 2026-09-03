// Package harness is the table of coding agents jstack installs into. Every
// harness name in the binary lives here; the skills and the letter stay
// harness-agnostic.
package harness

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Harness is one row of the table. Paths are relative to the home directory
// and use forward slashes.
type Harness struct {
	Key          string
	Name         string
	Detect       string
	Skills       string
	Instructions string
	Lead         string
}

const cursorLead = "---\ndescription: jstack, how the human you work for works\nalwaysApply: true\n---\n\n"

var table = []Harness{
	{Key: "claude", Name: "Claude Code", Detect: ".claude", Skills: ".claude/skills", Instructions: ".claude/CLAUDE.md"},
	{Key: "codex", Name: "Codex", Detect: ".codex", Skills: ".codex/skills", Instructions: ".codex/AGENTS.md"},
	{Key: "opencode", Name: "OpenCode", Detect: ".config/opencode", Skills: ".config/opencode/skills", Instructions: ".config/opencode/AGENTS.md"},
	{Key: "cursor", Name: "Cursor", Detect: ".cursor", Skills: ".cursor/skills", Instructions: ".cursor/rules/jstack.mdc", Lead: cursorLead},
	{Key: "pi", Name: "Pi", Detect: ".pi/agent", Skills: ".pi/agent/skills", Instructions: ".pi/agent/AGENTS.md"},
}

// All returns every row in table order.
func All() []Harness {
	return append([]Harness(nil), table...)
}

// Found returns the rows whose detect folder exists under home.
func Found(home string) []Harness {
	var found []Harness
	for _, entry := range table {
		if entry.Installed(home) {
			found = append(found, entry)
		}
	}
	return found
}

// Installed reports whether the harness's detect folder exists under home.
func (h Harness) Installed(home string) bool {
	info, err := os.Stat(h.DetectDir(home))
	return err == nil && info.IsDir()
}

// DetectDir is the absolute folder whose presence means the harness is installed.
func (h Harness) DetectDir(home string) string {
	return filepath.Join(home, filepath.FromSlash(h.Detect))
}

// SkillsDir is the absolute folder the harness loads skills from.
func (h Harness) SkillsDir(home string) string {
	return filepath.Join(home, filepath.FromSlash(h.Skills))
}

// InstructionsPath is the absolute file the harness reads on every turn.
func (h Harness) InstructionsPath(home string) string {
	return filepath.Join(home, filepath.FromSlash(h.Instructions))
}

// Parse turns a --harness value such as "claude,codex" or "all" into rows.
func Parse(spec string) ([]Harness, error) {
	spec = strings.TrimSpace(spec)
	if spec == "all" {
		return All(), nil
	}
	var keys []string
	for _, key := range strings.Split(spec, ",") {
		if key = strings.TrimSpace(key); key != "" {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("[JSTACK-HARNESS-EMPTY] --harness needs a value; expected a comma-separated list such as --harness claude,codex or --harness all; known keys: %s", strings.Join(Keys(All()), ", "))
	}
	return ByKeys(keys)
}

// ByKeys resolves keys to rows, keeping table order and dropping repeats.
func ByKeys(keys []string) ([]Harness, error) {
	wanted := map[string]bool{}
	for _, key := range keys {
		if _, ok := byKey(key); !ok {
			return nil, fmt.Errorf("[JSTACK-HARNESS-UNKNOWN] unknown harness %q; expected one of %s; example: --harness claude,codex", key, strings.Join(Keys(All()), ", "))
		}
		wanted[key] = true
	}
	var rows []Harness
	for _, entry := range table {
		if wanted[entry.Key] {
			rows = append(rows, entry)
		}
	}
	return rows, nil
}

// Keys lists the keys of rows, for flags and the config file.
func Keys(rows []Harness) []string {
	keys := make([]string, 0, len(rows))
	for _, entry := range rows {
		keys = append(keys, entry.Key)
	}
	return keys
}

func byKey(key string) (Harness, bool) {
	for _, entry := range table {
		if entry.Key == key {
			return entry, true
		}
	}
	return Harness{}, false
}
