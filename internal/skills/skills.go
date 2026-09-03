// Package skills copies the embedded skill folders into a harness's skills
// folder, backing up what it overwrites and never touching a skill it doesn't
// own.
package skills

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Plan groups the embedded skills by what applying them to dest would do.
// Local lists folders in dest that jstack doesn't own; they are never touched.
type Plan struct {
	New     []string
	Changed []string
	Same    []string
	Local   []string
}

// Pending reports whether applying the plan would change anything.
func (p Plan) Pending() bool {
	return len(p.New) > 0 || len(p.Changed) > 0
}

// Names lists the skill folders in source, the ones with a SKILL.md.
func Names(source fs.FS) ([]string, error) {
	entries, err := fs.ReadDir(source, ".")
	if err != nil {
		return nil, fmt.Errorf("[JSTACK-SKILLS-EMBED] cannot list the embedded skills: %w; this binary was built without its skills folder, reinstall it", err)
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := fs.Stat(source, entry.Name()+"/SKILL.md"); err == nil {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// PlanFor compares every embedded skill with its copy under dest.
func PlanFor(source fs.FS, dest string) (Plan, error) {
	names, err := Names(source)
	if err != nil {
		return Plan{}, err
	}
	owned := map[string]bool{}
	var plan Plan
	for _, name := range names {
		owned[name] = true
		target := filepath.Join(dest, name)
		info, statErr := os.Stat(target)
		switch {
		case statErr != nil || !info.IsDir():
			plan.New = append(plan.New, name)
		default:
			same, err := dirSame(source, name, target)
			if err != nil {
				return Plan{}, err
			}
			if same {
				plan.Same = append(plan.Same, name)
			} else {
				plan.Changed = append(plan.Changed, name)
			}
		}
	}
	entries, err := os.ReadDir(dest)
	if err != nil && !os.IsNotExist(err) {
		return Plan{}, fmt.Errorf("[JSTACK-SKILLS-DEST] cannot read the skills folder %q: %w; make it readable and rerun", dest, err)
	}
	for _, entry := range entries {
		if entry.IsDir() && !owned[entry.Name()] && !strings.HasPrefix(entry.Name(), ".") {
			plan.Local = append(plan.Local, entry.Name())
		}
	}
	return plan, nil
}

// Apply copies the new and changed skills into dest. A changed skill is moved
// to backupDir first, so nothing the machine had is lost.
func Apply(source fs.FS, dest string, plan Plan, backupDir string) error {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("[JSTACK-SKILLS-DEST] cannot create the skills folder %q: %w; make its parent writable and rerun", dest, err)
	}
	for _, name := range plan.Changed {
		if err := os.MkdirAll(backupDir, 0o755); err != nil {
			return fmt.Errorf("[JSTACK-SKILLS-BACKUP] cannot create the backup folder %q: %w; make it writable and rerun", backupDir, err)
		}
		if err := os.Rename(filepath.Join(dest, name), filepath.Join(backupDir, name)); err != nil {
			return fmt.Errorf("[JSTACK-SKILLS-BACKUP] cannot move %q into %q: %w; nothing was copied yet, fix the permissions and rerun", filepath.Join(dest, name), backupDir, err)
		}
	}
	for _, name := range append(append([]string{}, plan.New...), plan.Changed...) {
		if err := copyDir(source, name, filepath.Join(dest, name)); err != nil {
			return fmt.Errorf("[JSTACK-SKILLS-COPY] cannot write skill %q into %q: %w; the previous copy is in %q, fix the permissions and rerun", name, dest, err, backupDir)
		}
	}
	return nil
}

func dirSame(source fs.FS, name, target string) (bool, error) {
	embedded := map[string][]byte{}
	err := fs.WalkDir(source, name, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		content, err := fs.ReadFile(source, path)
		if err != nil {
			return err
		}
		embedded[strings.TrimPrefix(path, name+"/")] = content
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("[JSTACK-SKILLS-EMBED] cannot read embedded skill %q: %w; reinstall the binary", name, err)
	}
	seen := 0
	same := true
	err = filepath.WalkDir(target, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(entry.Name(), ".") && path != target {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		relative, _ := filepath.Rel(target, path)
		want, ok := embedded[filepath.ToSlash(relative)]
		if !ok {
			same = false
			return filepath.SkipAll
		}
		got, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !bytes.Equal(want, got) {
			same = false
			return filepath.SkipAll
		}
		seen++
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("[JSTACK-SKILLS-DEST] cannot read the installed skill %q: %w; make it readable and rerun", target, err)
	}
	return same && seen == len(embedded), nil
}

func copyDir(source fs.FS, name, target string) error {
	return fs.WalkDir(source, name, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative := strings.TrimPrefix(path, name)
		destination := filepath.Join(target, filepath.FromSlash(relative))
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		content, err := fs.ReadFile(source, path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, content, 0o644)
	})
}
