package setup

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// scriptsDir is where the install scripts that tools.md lines run land, so a
// line can run the script that shipped with this binary instead of reading
// one from the network.
func scriptsDir(home string) string {
	return filepath.Join(home, ".squirrel", "scripts")
}

// writeScripts puts every embedded script in scriptsDir before any tool line
// runs, replacing whatever an older binary left there.
func writeScripts(embedded assets, home string) error {
	entries, err := fs.ReadDir(embedded.scripts, ".")
	if err != nil {
		return fmt.Errorf("[SQUIRREL-EMBED] the binary has no scripts embedded: %w; reinstall it", err)
	}
	dir := scriptsDir(home)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("[SQUIRREL-SCRIPTS-WRITE] cannot create %q: %w; make the home folder writable and rerun", dir, err)
	}
	for _, entry := range entries {
		content, err := fs.ReadFile(embedded.scripts, entry.Name())
		if err != nil {
			return fmt.Errorf("[SQUIRREL-EMBED] cannot read the embedded script %q: %w; reinstall the binary", entry.Name(), err)
		}
		path := filepath.Join(dir, entry.Name())
		if err := os.WriteFile(path, content, 0o644); err != nil {
			return fmt.Errorf("[SQUIRREL-SCRIPTS-WRITE] cannot write %q: %w; make its folder writable and rerun", path, err)
		}
	}
	return nil
}
