package skills

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
