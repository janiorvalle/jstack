// Package skills copies skill folders from their sources into a harness's
// skills folder, backing up what it overwrites and never touching a skill no
// source owns.
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

// Source is one folder of skills: jstack's embedded folder, or the skills/
// folder of a repo the human named. Name is what the report calls it,
// "jstack" or "owner/name". Files holds one folder per skill at its root.
type Source struct {
	Name  string
	Files fs.FS
}

// Skill is one skill folder and the source it comes from.
type Skill struct {
	Name   string
	Source Source
}

// Collision is one skill name that more than one source holds, with the
// sources holding it in source order.
type Collision struct {
	Name    string
	Sources []string
}

// Plan groups the skills by what applying them to dest would do. Local lists
// folders in dest that no source owns; they are never touched.
type Plan struct {
	New     []Skill
	Changed []Skill
	Same    []Skill
	Local   []string
}

// Pending reports whether applying the plan would change anything.
func (p Plan) Pending() bool {
	return len(p.New) > 0 || len(p.Changed) > 0
}

// Names lists the skill folders in source, the ones with a SKILL.md.
func Names(source Source) ([]string, error) {
	entries, err := fs.ReadDir(source.Files, ".")
	if err != nil {
		return nil, fmt.Errorf("[JSTACK-SKILLS-SOURCE] cannot list the skills in %s: %w; %s", source.Name, err, sourceFix(source))
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := fs.Stat(source.Files, entry.Name()+"/SKILL.md"); err == nil {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// sourceFix is the next step when a source can't be read: the embedded
// folder means a broken binary, a repo means its clone under ~/.jstack.
func sourceFix(source Source) string {
	if source.Name == "jstack" {
		return "this binary was built without its skills folder, reinstall it"
	}
	return "its clone under ~/.jstack/repos is unreadable, or a file in the repo is a symlink pointing outside it; fix the repo or delete the clone and rerun"
}

// Collisions finds the names held by more than one source, sorted by name.
func Collisions(sources []Source) ([]Collision, error) {
	holders := map[string][]string{}
	for _, source := range sources {
		names, err := Names(source)
		if err != nil {
			return nil, err
		}
		for _, name := range names {
			holders[name] = append(holders[name], source.Name)
		}
	}
	var collisions []Collision
	for name, held := range holders {
		if len(held) > 1 {
			collisions = append(collisions, Collision{Name: name, Sources: held})
		}
	}
	sort.Slice(collisions, func(i, j int) bool { return collisions[i].Name < collisions[j].Name })
	return collisions, nil
}

// owners picks one source per skill name. A name held by several sources
// comes from the one picks names, or from the first source that holds it
// when picks has no entry.
func owners(sources []Source, picks map[string]string) ([]Skill, error) {
	bySource := map[string]Source{}
	for _, source := range sources {
		bySource[source.Name] = source
	}
	owned := map[string]Source{}
	var names []string
	for _, source := range sources {
		found, err := Names(source)
		if err != nil {
			return nil, err
		}
		for _, name := range found {
			if _, seen := owned[name]; !seen {
				names = append(names, name)
				owned[name] = source
			}
		}
	}
	for name, pick := range picks {
		if source, ok := bySource[pick]; ok {
			if _, held := fs.Stat(source.Files, name+"/SKILL.md"); held == nil {
				owned[name] = source
			}
		}
	}
	sort.Strings(names)
	skills := make([]Skill, 0, len(names))
	for _, name := range names {
		skills = append(skills, Skill{Name: name, Source: owned[name]})
	}
	return skills, nil
}

// PlanFor compares every skill in sources with its copy under dest. picks
// names the source for each name more than one source holds.
func PlanFor(sources []Source, picks map[string]string, dest string) (Plan, error) {
	skills, err := owners(sources, picks)
	if err != nil {
		return Plan{}, err
	}
	entries, err := os.ReadDir(dest)
	if err != nil && !os.IsNotExist(err) {
		return Plan{}, fmt.Errorf("[JSTACK-SKILLS-DEST] cannot read the skills folder %q: %w; make it readable and rerun", dest, err)
	}
	installed := map[string]bool{}
	for _, entry := range entries {
		installed[entry.Name()] = true
	}
	owned := map[string]bool{}
	var plan Plan
	for _, skill := range skills {
		owned[skill.Name] = true
		target := filepath.Join(dest, skill.Name)
		info, statErr := os.Stat(target)
		switch {
		case statErr != nil || !info.IsDir():
			plan.New = append(plan.New, skill)
		case !installed[skill.Name]:
			// The folder is there under another casing, so this disk
			// folds case and the two names are one folder. Installing
			// would overwrite a skill no source owns.
			return Plan{}, fmt.Errorf("[JSTACK-SKILLS-CASE] skill %q from %s and the local skill %q in %s are one folder on this disk, which ignores case; rename the local one and rerun", skill.Name, skill.Source.Name, caseTwin(entries, skill.Name), dest)
		default:
			same, err := dirSame(skill, target)
			if err != nil {
				return Plan{}, err
			}
			if same {
				plan.Same = append(plan.Same, skill)
			} else {
				plan.Changed = append(plan.Changed, skill)
			}
		}
	}
	for _, entry := range entries {
		if entry.IsDir() && !owned[entry.Name()] && !strings.HasPrefix(entry.Name(), ".") {
			plan.Local = append(plan.Local, entry.Name())
		}
	}
	return plan, nil
}

func caseTwin(entries []os.DirEntry, name string) string {
	for _, entry := range entries {
		if strings.EqualFold(entry.Name(), name) {
			return entry.Name()
		}
	}
	return name
}

// Apply puts the new and changed skills into dest one skill at a time. Each
// one is staged beside its destination first, then swapped in, so a copy that
// fails leaves that skill as it was and every earlier skill fully replaced.
// A changed skill is copied to backupDir before the swap.
func Apply(dest string, plan Plan, backupDir string) error {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("[JSTACK-SKILLS-DEST] cannot create the skills folder %q: %w; make its parent writable and rerun", dest, err)
	}
	for _, skill := range plan.New {
		if err := swapIn(skill, dest, false, backupDir); err != nil {
			return err
		}
	}
	for _, skill := range plan.Changed {
		if err := swapIn(skill, dest, true, backupDir); err != nil {
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
func swapIn(skill Skill, dest string, replace bool, backupDir string) error {
	name := skill.Name
	staged, err := os.MkdirTemp(dest, ".jstack-staging-"+name+"-")
	if err != nil {
		return fmt.Errorf("[JSTACK-SKILLS-COPY] cannot stage skill %q in %q: %w; make the folder writable and rerun", name, dest, err)
	}
	defer os.RemoveAll(staged)
	final := filepath.Join(dest, name)
	if err := copyDir(skill, staged); err != nil {
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
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return restore(retired, final, fmt.Errorf("[JSTACK-SKILLS-BACKUP] cannot create the backup folder %q: %w", backupDir, err))
	}
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
// symlinks and files and folders keep their modes, so the backup can be put
// back whole. Folders always get owner write and search so their contents
// can land in them.
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
			return os.MkdirAll(destination, info.Mode().Perm()|0o700)
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

func dirSame(skill Skill, target string) (bool, error) {
	name := skill.Name
	wanted := map[string][]byte{}
	err := fs.WalkDir(skill.Source.Files, name, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		content, err := fs.ReadFile(skill.Source.Files, path)
		if err != nil {
			return err
		}
		wanted[strings.TrimPrefix(path, name+"/")] = content
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("[JSTACK-SKILLS-SOURCE] cannot read skill %q from %s: %w; %s", name, skill.Source.Name, err, sourceFix(skill.Source))
	}
	seen := 0
	same := true
	err = filepath.WalkDir(target, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relative, _ := filepath.Rel(target, path)
		key := filepath.ToSlash(relative)
		want, ok := wanted[key]
		if !ok {
			if machineNoise(entry.Name()) {
				return nil
			}
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
	return same && seen == len(wanted), nil
}

// machineNoise is a file the desktop drops into folders it browses. It is
// the machine's, not drift; any other file the source lacks, hidden or not,
// makes the skill changed, so a file a repo deletes leaves the harness too.
func machineNoise(name string) bool {
	switch name {
	case ".DS_Store", "Thumbs.db", "desktop.ini", ".directory":
		return true
	}
	return strings.HasPrefix(name, "._")
}

func copyDir(skill Skill, target string) error {
	name := skill.Name
	return fs.WalkDir(skill.Source.Files, name, func(path string, entry fs.DirEntry, err error) error {
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
		content, err := fs.ReadFile(skill.Source.Files, path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, content, fileMode(content))
	})
}

// fileMode keeps skill scripts runnable. The embedded tree carries no modes
// and a clone's modes aren't trusted either; a file that starts with a shebang
// is one the skill tells the agent to run.
func fileMode(content []byte) os.FileMode {
	if bytes.HasPrefix(content, []byte("#!")) {
		return 0o755
	}
	return 0o644
}
