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

func source() []Source {
	return []Source{{Name: "squirrel", Files: squirrelFiles()}}
}

func squirrelFiles() fstest.MapFS {
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

// repo is a skills repo of the human's own: one skill of its own and one
// named the same as a squirrel skill.
func repo() Source {
	return Source{Name: "me/work-skills", Files: fstest.MapFS{
		"deploy/SKILL.md": {Data: []byte("deploy\n")},
		"voice/SKILL.md":  {Data: []byte("voice, my way\n")},
	}}
}

func names(skills []Skill) string {
	var parts []string
	for _, skill := range skills {
		parts = append(parts, skill.Name)
	}
	return strings.Join(parts, ",")
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
		if newPath == target && strings.Contains(oldPath, ".squirrel-staging-") {
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
		if strings.HasPrefix(entry.Name(), ".squirrel-") {
			t.Fatalf("work folder left behind in dest: %s", entry.Name())
		}
	}
}

func TestNamesAreFoldersWithASkillFile(t *testing.T) {
	names, err := Names(source()[0])
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
	write(t, filepath.Join(dest, ".squirrel-backup", "x"), "")
	plan, err := PlanFor(source(), nil, dest)
	if err != nil {
		t.Fatal(err)
	}
	if got := names(plan.New); got != "why" {
		t.Fatalf("new = %q", got)
	}
	if got := names(plan.Changed); got != "voice" {
		t.Fatalf("changed = %q", got)
	}
	if got := names(plan.Same); got != "how" {
		t.Fatalf("same = %q", got)
	}
	if got := strings.Join(plan.Local, ","); got != "mine" {
		t.Fatalf("local = %q", got)
	}
}

func TestSourceDotfilesArePartOfTheSkillAndMachineDotfilesAreNot(t *testing.T) {
	dest := t.TempDir()
	sources := []Source{{Name: "me/work-skills", Files: fstest.MapFS{
		"deploy/SKILL.md":         {Data: []byte("deploy\n")},
		"deploy/.env.example":     {Data: []byte("TOKEN=\n")},
		"deploy/.config/rules.md": {Data: []byte("rules\n")},
	}}}
	write(t, filepath.Join(dest, "deploy", "SKILL.md"), "deploy\n")
	write(t, filepath.Join(dest, "deploy", ".env.example"), "TOKEN=\n")
	write(t, filepath.Join(dest, "deploy", ".config", "rules.md"), "rules\n")
	write(t, filepath.Join(dest, "deploy", ".DS_Store"), "noise")
	write(t, filepath.Join(dest, "deploy", ".config", ".DS_Store"), "noise")
	plan, err := PlanFor(sources, nil, dest)
	if err != nil {
		t.Fatal(err)
	}
	if names(plan.Same) != "deploy" {
		t.Fatalf("an identical skill with a tracked dotfile is not same: %+v", plan)
	}
	write(t, filepath.Join(dest, "deploy", ".env.example"), "TOKEN=changed\n")
	plan, err = PlanFor(sources, nil, dest)
	if err != nil {
		t.Fatal(err)
	}
	if names(plan.Changed) != "deploy" {
		t.Fatalf("a changed tracked dotfile is not a change: %+v", plan)
	}
}

func TestDotfileTheSourceDroppedLeavesTheHarness(t *testing.T) {
	dest := t.TempDir()
	sources := []Source{{Name: "me/work-skills", Files: fstest.MapFS{"deploy/SKILL.md": {Data: []byte("deploy\n")}}}}
	write(t, filepath.Join(dest, "deploy", "SKILL.md"), "deploy\n")
	write(t, filepath.Join(dest, "deploy", ".env.example"), "TOKEN=\n")
	write(t, filepath.Join(dest, "deploy", ".DS_Store"), "noise")
	plan, err := PlanFor(sources, nil, dest)
	if err != nil {
		t.Fatal(err)
	}
	if names(plan.Changed) != "deploy" {
		t.Fatalf("a dotfile the source no longer has is not drift: %+v", plan)
	}
	if err := Apply(dest, plan, filepath.Join(t.TempDir(), "backup")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "deploy", ".env.example")); !os.IsNotExist(err) {
		t.Fatal("the dropped dotfile is still installed")
	}
	write(t, filepath.Join(dest, "deploy", ".DS_Store"), "noise")
	after, err := PlanFor(sources, nil, dest)
	if err != nil || names(after.Same) != "deploy" {
		t.Fatalf("machine noise counts as drift: %+v, %v", after, err)
	}
}

func TestLocalSkillDifferingOnlyInCaseIsNeverOverwritten(t *testing.T) {
	dest := t.TempDir()
	write(t, filepath.Join(dest, "Deploy", "SKILL.md"), "my local Deploy\n")
	source := []Source{{Name: "me/work-skills", Files: fstest.MapFS{"deploy/SKILL.md": {Data: []byte("deploy\n")}}}}
	if _, err := os.Stat(filepath.Join(dest, "deploy")); err != nil {
		plan, err := PlanFor(source, nil, dest)
		if err != nil || names(plan.New) != "deploy" || strings.Join(plan.Local, ",") != "Deploy" {
			t.Fatalf("case-sensitive disk: plan = %+v, %v", plan, err)
		}
		return
	}
	_, err := PlanFor(source, nil, dest)
	if err == nil || !strings.Contains(err.Error(), "SQUIRREL-SKILLS-CASE") || !strings.Contains(err.Error(), `local skill "Deploy"`) {
		t.Fatalf("case-insensitive disk: err = %v", err)
	}
}

func TestExtraFileMakesASkillChanged(t *testing.T) {
	dest := t.TempDir()
	write(t, filepath.Join(dest, "how", "SKILL.md"), "how\n")
	write(t, filepath.Join(dest, "how", "extra.md"), "added locally\n")
	plan, err := PlanFor(source(), nil, dest)
	if err != nil {
		t.Fatal(err)
	}
	if names(plan.Changed) != "how" {
		t.Fatalf("changed = %v", plan.Changed)
	}
}

func TestMissingDestIsAllNew(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "skills")
	plan, err := PlanFor(source(), nil, dest)
	if err != nil {
		t.Fatal(err)
	}
	if names(plan.New) != "how,voice,why" || len(plan.Local) != 0 {
		t.Fatalf("plan = %+v", plan)
	}
}

func TestCollisionsNameEverySourceHoldingASkill(t *testing.T) {
	sources := append(source(), repo(), Source{Name: "me/more", Files: fstest.MapFS{"voice/SKILL.md": {Data: []byte("third\n")}}})
	collisions, err := Collisions(sources)
	if err != nil {
		t.Fatal(err)
	}
	if len(collisions) != 1 || collisions[0].Name != "voice" || strings.Join(collisions[0].Sources, ",") != "squirrel,me/work-skills,me/more" {
		t.Fatalf("collisions = %+v", collisions)
	}
	if none, _ := Collisions(source()); len(none) != 0 {
		t.Fatalf("one source collides with itself: %+v", none)
	}
}

func TestPlanTakesACollidingSkillFromThePickedSource(t *testing.T) {
	dest := t.TempDir()
	write(t, filepath.Join(dest, "voice", "SKILL.md"), "voice v2\n")
	write(t, filepath.Join(dest, "voice", "notes.md"), "notes\n")
	sources := append(source(), repo())
	kept, err := PlanFor(sources, map[string]string{"voice": "squirrel"}, dest)
	if err != nil {
		t.Fatal(err)
	}
	if names(kept.Same) != "voice" || names(kept.New) != "deploy,how,why" {
		t.Fatalf("kept squirrel's: %+v", kept)
	}
	if kept.New[0].Source.Name != "me/work-skills" || kept.New[1].Source.Name != "squirrel" {
		t.Fatalf("sources = %s, %s", kept.New[0].Source.Name, kept.New[1].Source.Name)
	}
	overridden, err := PlanFor(sources, map[string]string{"voice": "me/work-skills"}, dest)
	if err != nil {
		t.Fatal(err)
	}
	if names(overridden.Changed) != "voice" || overridden.Changed[0].Source.Name != "me/work-skills" {
		t.Fatalf("overridden: %+v", overridden)
	}
	if err := Apply(dest, overridden, filepath.Join(t.TempDir(), "backup")); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(dest, "voice", "SKILL.md")); string(got) != "voice, my way\n" {
		t.Fatalf("installed voice = %q", got)
	}
	if _, err := os.Stat(filepath.Join(dest, "voice", "notes.md")); !os.IsNotExist(err) {
		t.Fatal("squirrel's notes.md survived the override")
	}
}

func TestSkillsFromASecondSourceAreOwnedNotLocal(t *testing.T) {
	dest := t.TempDir()
	write(t, filepath.Join(dest, "deploy", "SKILL.md"), "deploy\n")
	write(t, filepath.Join(dest, "mine", "SKILL.md"), "local skill\n")
	plan, err := PlanFor([]Source{source()[0], {Name: "me/work-skills", Files: fstest.MapFS{"deploy/SKILL.md": {Data: []byte("deploy\n")}}}}, nil, dest)
	if err != nil {
		t.Fatal(err)
	}
	if names(plan.Same) != "deploy" || strings.Join(plan.Local, ",") != "mine" {
		t.Fatalf("plan = %+v", plan)
	}
}

func TestUnreadableRepoSourceNamesTheClone(t *testing.T) {
	_, err := Names(Source{Name: "me/work-skills", Files: os.DirFS(filepath.Join(t.TempDir(), "missing"))})
	if err == nil || !strings.Contains(err.Error(), "SQUIRREL-SKILLS-SOURCE") || !strings.Contains(err.Error(), "me/work-skills") || !strings.Contains(err.Error(), "~/.squirrel/repos") {
		t.Fatalf("err = %v", err)
	}
}

func TestApplyCopiesNewBacksUpChangedAndLeavesLocalAlone(t *testing.T) {
	dest := t.TempDir()
	write(t, filepath.Join(dest, "voice", "SKILL.md"), "voice v1\n")
	write(t, filepath.Join(dest, "voice", "old-only.md"), "keep me in the backup\n")
	write(t, filepath.Join(dest, "mine", "SKILL.md"), "local skill\n")
	backup := filepath.Join(t.TempDir(), "backup", "claude", "skills")
	plan, err := PlanFor(source(), nil, dest)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(dest, plan, backup); err != nil {
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
	after, err := PlanFor(source(), nil, dest)
	if err != nil {
		t.Fatal(err)
	}
	if after.Pending() || names(after.Same) != "how,voice,why" {
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
	plan, err := PlanFor(source(), nil, dest)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(dest, plan, backup); err != nil {
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

func TestBackupKeepsSymlinksAndFileModes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need a privilege on windows")
	}
	dest := t.TempDir()
	write(t, filepath.Join(dest, "voice", "SKILL.md"), "voice v1\n")
	write(t, filepath.Join(dest, "voice", "run"), "no shebang, made runnable by hand\n")
	if err := os.Chmod(filepath.Join(dest, "voice", "run"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("SKILL.md", filepath.Join(dest, "voice", "link.md")); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dest, "voice", "private", "notes.md"), "mine\n")
	if err := os.Chmod(filepath.Join(dest, "voice", "private"), 0o700); err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(t.TempDir(), "backup")
	plan, err := PlanFor(source(), nil, dest)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(dest, plan, backup); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(filepath.Join(backup, "voice", "link.md"))
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("link.md in the backup = %v, %v; want a symlink", info, err)
	}
	if link, _ := os.Readlink(filepath.Join(backup, "voice", "link.md")); link != "SKILL.md" {
		t.Fatalf("link.md points at %q", link)
	}
	for path, want := range map[string]os.FileMode{
		filepath.Join(backup, "voice", "run"):     0o700,
		filepath.Join(backup, "voice", "private"): 0o700,
	} {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != want {
			t.Fatalf("%s mode = %v, %v; want %o", path, info.Mode().Perm(), err, want)
		}
	}
}

func TestSymlinkedSkillFolderIsBackedUpAsALink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need a privilege on windows")
	}
	dest := t.TempDir()
	elsewhere := filepath.Join(t.TempDir(), "dotfiles", "voice")
	write(t, filepath.Join(elsewhere, "SKILL.md"), "voice v1\n")
	if err := os.Symlink(elsewhere, filepath.Join(dest, "voice")); err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(t.TempDir(), "backup", "claude", "skills")
	plan, err := PlanFor(source(), nil, dest)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(dest, plan, backup); err != nil {
		t.Fatal(err)
	}
	if link, err := os.Readlink(filepath.Join(backup, "voice")); err != nil || link != elsewhere {
		t.Fatalf("backup voice = %q, %v; want a link to %q", link, err, elsewhere)
	}
	if got, _ := os.ReadFile(filepath.Join(dest, "voice", "SKILL.md")); string(got) != "voice v2\n" {
		t.Fatalf("installed voice = %q", got)
	}
	if got, _ := os.ReadFile(filepath.Join(elsewhere, "SKILL.md")); string(got) != "voice v1\n" {
		t.Fatalf("the linked folder was touched: %q", got)
	}
	assertNoWorkFolders(t, dest)
}

func TestApplyPutsTheOldSkillBackWhenTheSwapFails(t *testing.T) {
	dest := t.TempDir()
	write(t, filepath.Join(dest, "voice", "SKILL.md"), "voice v1\n")
	write(t, filepath.Join(dest, "voice", "old-only.md"), "keep me\n")
	failRenameInto(t, filepath.Join(dest, "voice"))
	backup := filepath.Join(t.TempDir(), "backup")
	plan, err := PlanFor(source(), nil, dest)
	if err != nil {
		t.Fatal(err)
	}
	err = Apply(dest, plan, backup)
	if err == nil || !strings.Contains(err.Error(), "SQUIRREL-SKILLS-COPY") || !strings.Contains(err.Error(), "back as it was") {
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
	plan, err := PlanFor(source(), nil, dest)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(dest, plan, backup); err != nil {
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
	plan, err := PlanFor(source(), nil, dest)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dest, "why", "mine.md"), "appeared after the plan\n")
	err = Apply(dest, plan, backup)
	if err == nil || !strings.Contains(err.Error(), "SQUIRREL-SKILLS-COPY") || !strings.Contains(err.Error(), `"why"`) {
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

// installedRoast is a tool's own skill as the tool left it on disk, the
// shape Sync reads from.
func installedRoast(t *testing.T, content string) Skill {
	t.Helper()
	folder := t.TempDir()
	write(t, filepath.Join(folder, "roast", "SKILL.md"), content)
	return Skill{Name: "roast", Source: Source{Name: "roast", Files: os.DirFS(folder)}}
}

func TestSyncCopiesASkillTheDestLacks(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "skills")
	changed, err := Sync(installedRoast(t, "roast v1\n"), dest, filepath.Join(t.TempDir(), "backup"))
	if err != nil {
		t.Fatal(err)
	}
	if content, _ := os.ReadFile(filepath.Join(dest, "roast", "SKILL.md")); !changed || string(content) != "roast v1\n" {
		t.Fatalf("changed = %v, roast = %q", changed, content)
	}
	assertNoWorkFolders(t, dest)
}

func TestSyncReplacesADifferentCopyAndBacksItUp(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "skills")
	backup := filepath.Join(t.TempDir(), "backup")
	write(t, filepath.Join(dest, "roast", "SKILL.md"), "roast v1\n")
	changed, err := Sync(installedRoast(t, "roast v2\n"), dest, backup)
	if err != nil {
		t.Fatal(err)
	}
	if content, _ := os.ReadFile(filepath.Join(dest, "roast", "SKILL.md")); !changed || string(content) != "roast v2\n" {
		t.Fatalf("changed = %v, roast = %q", changed, content)
	}
	if content, _ := os.ReadFile(filepath.Join(backup, "roast", "SKILL.md")); string(content) != "roast v1\n" {
		t.Fatalf("backup = %q", content)
	}
	assertNoWorkFolders(t, dest)
}

func TestSyncLeavesTheSameCopyAlone(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "skills")
	backup := filepath.Join(t.TempDir(), "backup")
	write(t, filepath.Join(dest, "roast", "SKILL.md"), "roast v1\n")
	changed, err := Sync(installedRoast(t, "roast v1\n"), dest, backup)
	if err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(backup); changed || statErr == nil {
		t.Fatalf("changed = %v, backup made = %v", changed, statErr == nil)
	}
}
