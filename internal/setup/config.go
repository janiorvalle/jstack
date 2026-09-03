package setup

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Config is what setup remembers between runs: the harnesses the human picked.
type Config struct {
	Harnesses []string `json:"harnesses"`
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
	if err := os.WriteFile(path, append(content, '\n'), 0o644); err != nil {
		return fmt.Errorf("[JSTACK-CONFIG-WRITE] cannot write %q: %w; make it writable and rerun", path, err)
	}
	return nil
}

// HasSavedPicks reports whether an earlier run saved harness picks, which is
// what lets a rerun apply without asking.
func HasSavedPicks(home string) bool {
	config, err := loadConfig(home)
	return err == nil && len(config.Harnesses) > 0
}
