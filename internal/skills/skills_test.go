package skills

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"testing/fstest"
)

func source() fstest.MapFS {
	return fstest.MapFS{
		"how/SKILL.md":    {Data: []byte("how\n")},
		"voice/SKILL.md":  {Data: []byte("voice v2\n")},
		"voice/notes.md":  {Data: []byte("notes\n")},
		"notes.md":        {Data: []byte("not a skill\n")},
		"folder/README":   {Data: []byte("no SKILL.md, not a skill\n")},
		"why/SKILL.md":    {Data: []byte("why\n")},
		"why/refs/one.md": {Data: []byte("one\n")},
		"why/scan.sh":     {Data: []byte("#!/bin/sh\necho scan\n")},
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// failRenamesOutside makes every rename that leaves or enters tree fail the way the
// kernel does across filesystems, which is what a CODEX_HOME or
// CLAUDE_CONFIG_DIR on another mount looks like to os.Rename.
func failRenamesOutside(t *testing.T, tree string) {
	t.Helper()
	rename = func(oldPath, newPath string) error {
		if !strings.HasPrefix(oldPath, tree) || !strings.HasPrefix(newPath, tree) {
			return &os.LinkError{Op: "rename", Old: oldPath, New: newPath, Err: syscall.EXDEV}
		}
		return os.Rename(oldPath, newPath)
	}
	t.Cleanup(func() { rename = os.Rename })
}

func failRenameInto(t *testing.T, target string) {
	t.Helper()
	rename = func(oldPath, newPath string) error {
		if newPath == target && strings.Contains(oldPath, ".jstack-staging-") {
			return &os.LinkError{Op: "rename", Old: oldPath, New: newPath, Err: syscall.EACCES}
		}
		return os.Rename(oldPath, newPath)
	}
	t.Cleanup(func() { rename = os.Rename })
}

func assertNoWorkFolders(t *testing.T, dest string) {
	t.Helper()
	entries, err := os.ReadDir(dest)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".jstack-") {
			t.Fatalf("work folder left behind in dest: %s", entry.Name())
		}
	}
}

func TestNamesAreFoldersWithASkillFile(t *testing.T) {
	names, err := Names(source())
	if err != nil || strings.Join(names, ",") != "how,voice,why" {
		t.Fatalf("names = %v, %v", names, err)
	}
}

func TestPlanGroupsByWhatWouldChange(t *testing.T) {
	dest := t.TempDir()
	write(t, filepath.Join(dest, "how", "SKILL.md"), "how\n")
	write(t, filepath.Join(dest, "how", ".DS_Store"), "noise")
	write(t, filepath.Join(dest, "voice", "SKILL.md"), "voice v1\n")
	write(t, filepath.Join(dest, "voice", "notes.md"), "notes\n")
	write(t, filepath.Join(dest, "mine", "SKILL.md"), "local skill\n")
	write(t, filepath.Join(dest, ".jstack-backup", "x"), "")
	plan, err := PlanFor(source(), dest)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(plan.New, ","); got != "why" {
		t.Fatalf("new = %q", got)
	}
	if got := strings.Join(plan.Changed, ","); got != "voice" {
		t.Fatalf("changed = %q", got)
	}
	if got := strings.Join(plan.Same, ","); got != "how" {
		t.Fatalf("same = %q", got)
	}
	if got := strings.Join(plan.Local, ","); got != "mine" {
		t.Fatalf("local = %q", got)
	}
}

func TestExtraFileMakesASkillChanged(t *testing.T) {
	dest := t.TempDir()
	write(t, filepath.Join(dest, "how", "SKILL.md"), "how\n")
	write(t, filepath.Join(dest, "how", "extra.md"), "added locally\n")
	plan, err := PlanFor(source(), dest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(plan.Changed, ",") != "how" {
		t.Fatalf("changed = %v", plan.Changed)
	}
}

func TestMissingDestIsAllNew(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "skills")
	plan, err := PlanFor(source(), dest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(plan.New, ",") != "how,voice,why" || len(plan.Local) != 0 {
		t.Fatalf("plan = %+v", plan)
	}
}

func TestApplyCopiesNewBacksUpChangedAndLeavesLocalAlone(t *testing.T) {
	dest := t.TempDir()
	write(t, filepath.Join(dest, "voice", "SKILL.md"), "voice v1\n")
	write(t, filepath.Join(dest, "voice", "old-only.md"), "keep me in the backup\n")
	write(t, filepath.Join(dest, "mine", "SKILL.md"), "local skill\n")
	backup := filepath.Join(t.TempDir(), "backup", "claude", "skills")
	plan, err := PlanFor(source(), dest)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(source(), dest, plan, backup); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]string{
		filepath.Join(dest, "voice", "SKILL.md"):      "voice v2\n",
		filepath.Join(dest, "why", "refs", "one.md"):  "one\n",
		filepath.Join(dest, "mine", "SKILL.md"):       "local skill\n",
		filepath.Join(backup, "voice", "SKILL.md"):    "voice v1\n",
		filepath.Join(backup, "voice", "old-only.md"): "keep me in the backup\n",
	} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Fatalf("%s = %q, %v; want %q", path, got, err, want)
		}
	}
	if _, err := os.Stat(filepath.Join(dest, "voice", "old-only.md")); !os.IsNotExist(err) {
		t.Fatalf("old-only.md survived in dest: %v", err)
	}
	if runtime.GOOS != "windows" {
		for path, want := range map[string]os.FileMode{
			filepath.Join(dest, "why", "scan.sh"):  0o755,
			filepath.Join(dest, "why", "SKILL.md"): 0o644,
		} {
			info, err := os.Stat(path)
			if err != nil || info.Mode().Perm() != want {
				t.Fatalf("%s mode = %v, %v; want %o", path, info.Mode().Perm(), err, want)
			}
		}
	}
	after, err := PlanFor(source(), dest)
	if err != nil {
		t.Fatal(err)
	}
	if after.Pending() || strings.Join(after.Same, ",") != "how,voice,why" {
		t.Fatalf("after = %+v", after)
	}
	assertNoWorkFolders(t, dest)
}

func TestApplyBacksUpAChangedSkillAcrossFilesystems(t *testing.T) {
	dest := t.TempDir()
	failRenamesOutside(t, dest)
	write(t, filepath.Join(dest, "voice", "SKILL.md"), "voice v1\n")
	write(t, filepath.Join(dest, "voice", "old-only.md"), "keep me in the backup\n")
	backup := filepath.Join(t.TempDir(), "backup", "codex", "skills")
	plan, err := PlanFor(source(), dest)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(source(), dest, plan, backup); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]string{
		filepath.Join(dest, "voice", "SKILL.md"):      "voice v2\n",
		filepath.Join(backup, "voice", "SKILL.md"):    "voice v1\n",
		filepath.Join(backup, "voice", "old-only.md"): "keep me in the backup\n",
	} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Fatalf("%s = %q, %v; want %q", path, got, err, want)
		}
	}
	assertNoWorkFolders(t, dest)
}

func TestApplyPutsTheOldSkillBackWhenTheSwapFails(t *testing.T) {
	dest := t.TempDir()
	write(t, filepath.Join(dest, "voice", "SKILL.md"), "voice v1\n")
	write(t, filepath.Join(dest, "voice", "old-only.md"), "keep me\n")
	failRenameInto(t, filepath.Join(dest, "voice"))
	backup := filepath.Join(t.TempDir(), "backup")
	plan, err := PlanFor(source(), dest)
	if err != nil {
		t.Fatal(err)
	}
	err = Apply(source(), dest, plan, backup)
	if err == nil || !strings.Contains(err.Error(), "JSTACK-SKILLS-COPY") || !strings.Contains(err.Error(), "back as it was") {
		t.Fatalf("err = %v", err)
	}
	for path, want := range map[string]string{
		filepath.Join(dest, "voice", "SKILL.md"):    "voice v1\n",
		filepath.Join(dest, "voice", "old-only.md"): "keep me\n",
		filepath.Join(backup, "voice", "SKILL.md"):  "voice v1\n",
	} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Fatalf("%s = %q, %v; want %q", path, got, err, want)
		}
	}
	assertNoWorkFolders(t, dest)
}

func TestApplyWithNothingChangedMakesNoBackup(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "skills")
	backup := filepath.Join(t.TempDir(), "backup")
	plan, err := PlanFor(source(), dest)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(source(), dest, plan, backup); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(backup); !os.IsNotExist(err) {
		t.Fatalf("backup folder exists: %v", err)
	}
}

func TestApplyFailureLeavesEachSkillWholeAndNoStaging(t *testing.T) {
	dest := t.TempDir()
	write(t, filepath.Join(dest, "voice", "SKILL.md"), "voice v1\n")
	backup := filepath.Join(t.TempDir(), "backup")
	plan, err := PlanFor(source(), dest)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dest, "why", "mine.md"), "appeared after the plan\n")
	err = Apply(source(), dest, plan, backup)
	if err == nil || !strings.Contains(err.Error(), "JSTACK-SKILLS-COPY") || !strings.Contains(err.Error(), `"why"`) {
		t.Fatalf("err = %v", err)
	}
	if got, _ := os.ReadFile(filepath.Join(dest, "voice", "SKILL.md")); string(got) != "voice v1\n" {
		t.Fatalf("voice changed although the run stopped on why: %q", got)
	}
	if _, err := os.Stat(backup); !os.IsNotExist(err) {
		t.Fatalf("a backup was made although nothing was swapped: %v", err)
	}
	if got, _ := os.ReadFile(filepath.Join(dest, "why", "mine.md")); string(got) != "appeared after the plan\n" {
		t.Fatalf("why was touched: %q", got)
	}
	assertNoWorkFolders(t, dest)
}
