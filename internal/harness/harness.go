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

// row is a table entry before it meets a machine. homeVar is the variable the
// harness's own docs name as moving its folder; a row gets one only when such
// docs exist.
type row struct {
	key          string
	name         string
	homeVar      string
	root         string
	skills       string
	instructions string
	lead         string
}

const cursorLead = "---\ndescription: jstack, how the human you work for works\nalwaysApply: true\n---\n\n"

var table = []row{
	{key: "claude", name: "Claude Code", homeVar: "CLAUDE_CONFIG_DIR", root: ".claude", skills: "skills", instructions: "CLAUDE.md"},
	{key: "codex", name: "Codex", homeVar: "CODEX_HOME", root: ".codex", skills: "skills", instructions: "AGENTS.md"},
	{key: "opencode", name: "OpenCode", root: ".config/opencode", skills: "skills", instructions: "AGENTS.md"},
	{key: "cursor", name: "Cursor", root: ".cursor", skills: "skills", instructions: "rules/jstack.mdc", lead: cursorLead},
	{key: "pi", name: "Pi", root: ".pi/agent", skills: "skills", instructions: "AGENTS.md"},
}

// Harness is one row resolved for a machine. Root is the absolute folder the
// harness reads from: the value of HomeVar when that variable is set and
// non-empty, otherwise the folder under home. HomeVar is empty for a row
// whose folder came from home.
type Harness struct {
	Key     string
	Name    string
	Root    string
	HomeVar string
	Lead    string

	skills       string
	instructions string
}

// Table is every harness resolved for one machine, in table order.
type Table []Harness

// Resolve builds the table for a machine. getenv answers the variables the
// rows document.
func Resolve(home string, getenv func(string) string) Table {
	rows := make(Table, 0, len(table))
	for _, entry := range table {
		rows = append(rows, entry.resolve(home, getenv))
	}
	return rows
}

func (r row) resolve(home string, getenv func(string) string) Harness {
	resolved := Harness{Key: r.key, Name: r.name, Lead: r.lead, skills: r.skills, instructions: r.instructions}
	resolved.Root = filepath.Join(home, filepath.FromSlash(r.root))
	if r.homeVar != "" && getenv(r.homeVar) != "" {
		resolved.Root = filepath.Clean(getenv(r.homeVar))
		resolved.HomeVar = r.homeVar
	}
	return resolved
}

// Found returns the rows whose folder exists.
func (t Table) Found() []Harness {
	var found []Harness
	for _, entry := range t {
		if entry.Installed() {
			found = append(found, entry)
		}
	}
	return found
}

// Installed reports whether the harness's folder exists.
func (h Harness) Installed() bool {
	info, err := os.Stat(h.Root)
	return err == nil && info.IsDir()
}

// SkillsDir is the absolute folder the harness loads skills from.
func (h Harness) SkillsDir() string {
	return filepath.Join(h.Root, filepath.FromSlash(h.skills))
}

// InstructionsPath is the absolute file the harness reads on every turn.
func (h Harness) InstructionsPath() string {
	return filepath.Join(h.Root, filepath.FromSlash(h.instructions))
}

// Parse turns a --harness value such as "claude,codex" or "all" into rows.
func (t Table) Parse(spec string) ([]Harness, error) {
	spec = strings.TrimSpace(spec)
	if spec == "all" {
		return append([]Harness(nil), t...), nil
	}
	var keys []string
	for _, key := range strings.Split(spec, ",") {
		if key = strings.TrimSpace(key); key != "" {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("[JSTACK-HARNESS-EMPTY] --harness needs a value; expected a comma-separated list such as --harness claude,codex or --harness all; known keys: %s", strings.Join(Keys(t), ", "))
	}
	return t.ByKeys(keys)
}

// ByKeys resolves keys to rows, keeping table order and dropping repeats.
func (t Table) ByKeys(keys []string) ([]Harness, error) {
	wanted := map[string]bool{}
	for _, key := range keys {
		if !t.has(key) {
			return nil, fmt.Errorf("[JSTACK-HARNESS-UNKNOWN] unknown harness %q; expected one of %s; example: --harness claude,codex", key, strings.Join(Keys(t), ", "))
		}
		wanted[key] = true
	}
	var rows []Harness
	for _, entry := range t {
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

func (t Table) has(key string) bool {
	for _, entry := range t {
		if entry.Key == key {
			return true
		}
	}
	return false
}
