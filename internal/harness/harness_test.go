package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func noEnv(string) string { return "" }

func env(values map[string]string) func(string) string {
	return func(name string) string { return values[name] }
}

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
	got := strings.Join(Keys(Resolve(home, noEnv).Found()), ",")
	if got != "claude,pi" {
		t.Fatalf("Found = %q, want claude,pi", got)
	}
}

func TestTableRowsPointUnderHomeWithoutTheVariables(t *testing.T) {
	home := filepath.Join("home", "me")
	for _, entry := range Resolve(home, noEnv) {
		for _, path := range []string{entry.Root, entry.SkillsDir(), entry.InstructionsPath()} {
			if !strings.HasPrefix(path, home+string(filepath.Separator)) {
				t.Fatalf("%s: %q is not under home", entry.Key, path)
			}
		}
		if !strings.HasPrefix(entry.SkillsDir(), entry.Root) {
			t.Fatalf("%s: skills %q not under root %q", entry.Key, entry.SkillsDir(), entry.Root)
		}
		if entry.HomeVar != "" {
			t.Fatalf("%s: HomeVar = %q with no variable set", entry.Key, entry.HomeVar)
		}
	}
}

func TestVariablesMoveTheirRowsAndNothingElse(t *testing.T) {
	home := filepath.Join("home", "me")
	codexHome := filepath.Join("work", "codex")
	claudeHome := filepath.Join("work", "claude")
	rows := Resolve(home, env(map[string]string{"CODEX_HOME": codexHome + "/", "CLAUDE_CONFIG_DIR": claudeHome}))
	want := map[string][3]string{
		"claude":   {claudeHome, filepath.Join(claudeHome, "skills"), filepath.Join(claudeHome, "CLAUDE.md")},
		"codex":    {codexHome, filepath.Join(codexHome, "skills"), filepath.Join(codexHome, "AGENTS.md")},
		"opencode": {filepath.Join(home, ".config", "opencode"), filepath.Join(home, ".config", "opencode", "skills"), filepath.Join(home, ".config", "opencode", "AGENTS.md")},
		"cursor":   {filepath.Join(home, ".cursor"), filepath.Join(home, ".cursor", "skills"), filepath.Join(home, ".cursor", "rules", "jstack.mdc")},
		"pi":       {filepath.Join(home, ".pi", "agent"), filepath.Join(home, ".pi", "agent", "skills"), filepath.Join(home, ".pi", "agent", "AGENTS.md")},
	}
	for _, entry := range rows {
		got := [3]string{entry.Root, entry.SkillsDir(), entry.InstructionsPath()}
		if got != want[entry.Key] {
			t.Fatalf("%s: paths = %v, want %v", entry.Key, got, want[entry.Key])
		}
	}
	vars := map[string]string{}
	for _, entry := range rows {
		vars[entry.Key] = entry.HomeVar
	}
	if vars["claude"] != "CLAUDE_CONFIG_DIR" || vars["codex"] != "CODEX_HOME" || vars["opencode"] != "" || vars["cursor"] != "" || vars["pi"] != "" {
		t.Fatalf("HomeVar by key = %v", vars)
	}
}

func TestEmptyVariableMeansTheDefault(t *testing.T) {
	home := filepath.Join("home", "me")
	rows := Resolve(home, env(map[string]string{"CODEX_HOME": "", "CLAUDE_CONFIG_DIR": ""}))
	for _, entry := range rows {
		if !strings.HasPrefix(entry.Root, home) || entry.HomeVar != "" {
			t.Fatalf("%s: root %q, HomeVar %q", entry.Key, entry.Root, entry.HomeVar)
		}
	}
}

func TestFoundLooksWhereTheVariablePoints(t *testing.T) {
	home := t.TempDir()
	codexHome := filepath.Join(t.TempDir(), "codex")
	for _, folder := range []string{filepath.Join(home, ".codex"), codexHome} {
		if err := os.MkdirAll(folder, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	found := Resolve(home, env(map[string]string{"CODEX_HOME": codexHome})).Found()
	if len(found) != 1 || found[0].Key != "codex" || found[0].Root != codexHome {
		t.Fatalf("Found = %+v", found)
	}
	if found := Resolve(home, env(map[string]string{"CODEX_HOME": filepath.Join(codexHome, "missing")})).Found(); len(found) != 0 {
		t.Fatalf("Found = %v with the variable pointing at a missing folder, want none", Keys(found))
	}
}

func TestCursorRuleCarriesAlwaysApplyFrontmatter(t *testing.T) {
	rows := Resolve("h", noEnv)
	picked, err := rows.ByKeys([]string{"cursor"})
	if err != nil {
		t.Fatal(err)
	}
	cursor := picked[0]
	if !strings.HasSuffix(filepath.ToSlash(cursor.InstructionsPath()), ".cursor/rules/jstack.mdc") {
		t.Fatalf("instructions = %q", cursor.InstructionsPath())
	}
	if !strings.HasPrefix(cursor.Lead, "---\n") || !strings.Contains(cursor.Lead, "alwaysApply: true") {
		t.Fatalf("lead = %q", cursor.Lead)
	}
	for _, entry := range rows {
		if entry.Key != "cursor" && entry.Lead != "" {
			t.Fatalf("%s has a lead", entry.Key)
		}
	}
}

func TestParseFlagValues(t *testing.T) {
	rows := Resolve("h", noEnv)
	all, err := rows.Parse("all")
	if err != nil || len(all) != len(rows) {
		t.Fatalf("all = %v, %v", Keys(all), err)
	}
	some, err := rows.Parse(" codex , claude,codex")
	if err != nil || strings.Join(Keys(some), ",") != "claude,codex" {
		t.Fatalf("list = %v, %v", Keys(some), err)
	}
	if _, err := rows.Parse("claude,emacs"); err == nil || !strings.Contains(err.Error(), "JSTACK-HARNESS-UNKNOWN") || !strings.Contains(err.Error(), "emacs") {
		t.Fatalf("unknown err = %v", err)
	}
	if _, err := rows.Parse(" , "); err == nil || !strings.Contains(err.Error(), "JSTACK-HARNESS-EMPTY") {
		t.Fatalf("empty err = %v", err)
	}
}
