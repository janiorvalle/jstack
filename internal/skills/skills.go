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
	"strconv"
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

// Apply puts the new and changed skills into dest one skill at a time. Each
// one is staged beside its destination first, then swapped in, so a copy that
// fails leaves that skill as it was and every earlier skill fully replaced.
// A changed skill is copied to backupDir before the swap.
func Apply(source fs.FS, dest string, plan Plan, backupDir string) error {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("[JSTACK-SKILLS-DEST] cannot create the skills folder %q: %w; make its parent writable and rerun", dest, err)
	}
	changed := map[string]bool{}
	for _, name := range plan.Changed {
		changed[name] = true
	}
	for _, name := range append(append([]string{}, plan.New...), plan.Changed...) {
		if err := swapIn(source, dest, name, changed[name], backupDir); err != nil {
			return err
		}
	}
	return nil
}

// rename is os.Rename behind a name the tests can point at a failing stand-in.
var rename = os.Rename

// swapIn only ever renames inside dest. The backup folder can sit on another
// filesystem, where a rename fails with EXDEV, so the old folder is retired
// beside its replacement first, copied from there into the backup, and only
// removed once the swap has succeeded.
func swapIn(source fs.FS, dest, name string, replace bool, backupDir string) error {
	staged, err := os.MkdirTemp(dest, ".jstack-staging-"+name+"-")
	if err != nil {
		return fmt.Errorf("[JSTACK-SKILLS-COPY] cannot stage skill %q in %q: %w; make the folder writable and rerun", name, dest, err)
	}
	defer os.RemoveAll(staged)
	final := filepath.Join(dest, name)
	if err := copyDir(source, name, staged); err != nil {
		return fmt.Errorf("[JSTACK-SKILLS-COPY] cannot write skill %q into %q: %w; the installed copy is untouched, fix the permissions or free space and rerun", name, dest, err)
	}
	if !replace {
		if err := rename(staged, final); err != nil {
			return fmt.Errorf("[JSTACK-SKILLS-COPY] cannot put skill %q in place at %q: %w; fix the permissions and rerun", name, final, err)
		}
		return nil
	}
	retired := filepath.Join(dest, ".jstack-retired-"+name+"-"+strconv.Itoa(os.Getpid()))
	if err := rename(final, retired); err != nil {
		return fmt.Errorf("[JSTACK-SKILLS-COPY] cannot move skill %q aside to %q: %w; the installed copy is untouched, fix the permissions and rerun", name, retired, err)
	}
	backup := filepath.Join(backupDir, name)
	if err := copyTree(retired, backup); err != nil {
		return restore(retired, final, fmt.Errorf("[JSTACK-SKILLS-BACKUP] cannot copy skill %q into %q: %w", name, backup, err))
	}
	if err := rename(staged, final); err != nil {
		return restore(retired, final, fmt.Errorf("[JSTACK-SKILLS-COPY] cannot put skill %q in place at %q: %w", name, final, err))
	}
	if err := os.RemoveAll(retired); err != nil {
		return fmt.Errorf("[JSTACK-SKILLS-CLEANUP] cannot remove the retired copy of skill %q at %q: %w; the new skill is in place and the old one is backed up in %q, delete that folder by hand", name, retired, err, backup)
	}
	return nil
}

// restore puts the retired folder back where it was and finishes the error
// message with what state the skill is in.
func restore(retired, final string, cause error) error {
	if err := rename(retired, final); err != nil {
		return fmt.Errorf("%w, and restoring it failed too: %v; move %q back to %q by hand", cause, err, retired, final)
	}
	return fmt.Errorf("%w; the installed copy is back as it was, fix the permissions and rerun", cause)
}

// copyTree copies an installed skill as it is on disk: symlinks stay
// symlinks and files keep their modes, so the backup can be put back whole.
func copyTree(source, target string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, _ := filepath.Rel(source, path)
		destination := filepath.Join(target, relative)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		switch {
		case entry.IsDir():
			return os.MkdirAll(destination, 0o755)
		case info.Mode()&os.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, destination)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, content, info.Mode().Perm())
	})
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
			if path == name {
				return nil
			}
			return os.MkdirAll(destination, 0o755)
		}
		content, err := fs.ReadFile(source, path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, content, fileMode(content))
	})
}

// fileMode keeps skill scripts runnable. The embedded tree carries no modes,
// and a file that starts with a shebang is one the skill tells the agent to run.
func fileMode(content []byte) os.FileMode {
	if bytes.HasPrefix(content, []byte("#!")) {
		return 0o755
	}
	return 0o644
}
