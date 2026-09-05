package setup

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Config is what setup remembers between runs: the harnesses the human
// picked, every harness found on any run so far, so one found since is
// offered checked while one they left out stays out, the skills
// repos they named, whether they were asked for one, which source each
// colliding skill name comes from, "jstack" or "owner/name", and the
// folders their repos live in, with whether they were asked for those.
type Config struct {
	Harnesses       []string          `json:"harnesses"`
	HarnessesFound  []string          `json:"harnesses_found,omitempty"`
	SkillRepos      []string          `json:"skill_repos,omitempty"`
	SkillReposAsked bool              `json:"skill_repos_asked,omitempty"`
	SkillOverrides  map[string]string `json:"skill_overrides,omitempty"`
	ReposDirs       []string          `json:"repos_dirs,omitempty"`
	ReposDirsAsked  bool              `json:"repos_dirs_asked,omitempty"`
}

func configPath(home string) string {
	return filepath.Join(home, ".jstack", "config.json")
}

func loadConfig(home string) (Config, error) {
	content, err := os.ReadFile(configPath(home))
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("[JSTACK-CONFIG-READ] cannot read %q: %w; make it readable or delete it and rerun", configPath(home), err)
	}
	var config Config
	if err := json.Unmarshal(content, &config); err != nil {
		return Config{}, fmt.Errorf("[JSTACK-CONFIG-PARSE] %q is not valid JSON: %w; expected {\"harnesses\":[\"claude\",\"codex\"]}; fix it or delete it and rerun", configPath(home), err)
	}
	return config, nil
}

func saveConfig(home string, config Config) error {
	path := configPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("[JSTACK-CONFIG-WRITE] cannot create %q: %w; make the home folder writable and rerun", filepath.Dir(path), err)
	}
	content, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".config-*")
	if err != nil {
		return fmt.Errorf("[JSTACK-CONFIG-WRITE] cannot stage %q: %w; make its folder writable and rerun", path, err)
	}
	defer os.Remove(temporary.Name())
	if _, err := temporary.Write(append(content, '\n')); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("[JSTACK-CONFIG-WRITE] cannot write %q: %w; make it writable and rerun", path, err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("[JSTACK-CONFIG-WRITE] cannot write %q: %w; make it writable and rerun", path, err)
	}
	if err := os.Rename(temporary.Name(), path); err != nil {
		return fmt.Errorf("[JSTACK-CONFIG-WRITE] cannot replace %q: %w; make it writable and rerun", path, err)
	}
	return nil
}

// HasSavedPicks reports whether an earlier run saved harness picks, which is
// what lets a rerun apply without asking.
func HasSavedPicks(home string) bool {
	config, err := loadConfig(home)
	return err == nil && len(config.Harnesses) > 0
}
