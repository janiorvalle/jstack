package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFoundDetectsByFolderInTableOrder(t *testing.T) {
	home := t.TempDir()
	for _, folder := range []string{".pi/agent", ".claude"} {
		if err := os.MkdirAll(filepath.Join(home, folder), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(home, ".codex"), []byte("a file, not a folder"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(Keys(Found(home)), ",")
	if got != "claude,pi" {
		t.Fatalf("Found = %q, want claude,pi", got)
	}
}

func TestTableRowsPointUnderHome(t *testing.T) {
	home := filepath.Join("home", "me")
	for _, entry := range All() {
		for _, path := range []string{entry.DetectDir(home), entry.SkillsDir(home), entry.InstructionsPath(home)} {
			if !strings.HasPrefix(path, home+string(filepath.Separator)) {
				t.Fatalf("%s: %q is not under home", entry.Key, path)
			}
		}
		if !strings.HasPrefix(entry.SkillsDir(home), entry.DetectDir(home)) {
			t.Fatalf("%s: skills %q not under detect %q", entry.Key, entry.SkillsDir(home), entry.DetectDir(home))
		}
	}
}

func TestCursorRuleCarriesAlwaysApplyFrontmatter(t *testing.T) {
	rows, err := ByKeys([]string{"cursor"})
	if err != nil {
		t.Fatal(err)
	}
	cursor := rows[0]
	if !strings.HasSuffix(filepath.ToSlash(cursor.InstructionsPath("h")), ".cursor/rules/jstack.mdc") {
		t.Fatalf("instructions = %q", cursor.InstructionsPath("h"))
	}
	if !strings.HasPrefix(cursor.Lead, "---\n") || !strings.Contains(cursor.Lead, "alwaysApply: true") {
		t.Fatalf("lead = %q", cursor.Lead)
	}
	for _, entry := range All() {
		if entry.Key != "cursor" && entry.Lead != "" {
			t.Fatalf("%s has a lead", entry.Key)
		}
	}
}

func TestParseFlagValues(t *testing.T) {
	all, err := Parse("all")
	if err != nil || len(all) != len(All()) {
		t.Fatalf("all = %v, %v", Keys(all), err)
	}
	some, err := Parse(" codex , claude,codex")
	if err != nil || strings.Join(Keys(some), ",") != "claude,codex" {
		t.Fatalf("list = %v, %v", Keys(some), err)
	}
	if _, err := Parse("claude,emacs"); err == nil || !strings.Contains(err.Error(), "JSTACK-HARNESS-UNKNOWN") || !strings.Contains(err.Error(), "emacs") {
		t.Fatalf("unknown err = %v", err)
	}
	if _, err := Parse(" , "); err == nil || !strings.Contains(err.Error(), "JSTACK-HARNESS-EMPTY") {
		t.Fatalf("empty err = %v", err)
	}
}
