package setup

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const workSkills = "me/work-skills"

// withRepo is a machine where gh can reach one skills repo of the human's
// own: a skill jstack doesn't have and one named the same as jstack's voice.
func withRepo() *fakeShell {
	shell := withRoast("1.1.0")
	shell.repos = map[string]map[string]string{workSkills: {
		"skills/deploy/SKILL.md": "deploy\n",
		"skills/voice/SKILL.md":  "voice, my way\n",
		"README.md":              "not a skill\n",
	}}
	return shell
}

func expectAll(t *testing.T, output string, expected ...string) {
	t.Helper()
	for _, want := range expected {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestRepoNameTakesWhatPeoplePaste(t *testing.T) {
	for _, spec := range []string{"me/work-skills", " me/work-skills ", "https://github.com/me/work-skills", "github.com/me/work-skills.git", "me/work-skills/"} {
		if name, err := repoName(spec); err != nil || name != workSkills {
			t.Fatalf("%q: name = %q, %v", spec, name, err)
		}
	}
	for _, spec := range []string{"", "work-skills", "me/", "/work-skills", "me/work skills", "me/a/b", "me/work;rm"} {
		if _, err := repoName(spec); err == nil || !strings.Contains(err.Error(), "JSTACK-SKILL-REPO") {
			t.Fatalf("%q: err = %v", spec, err)
		}
	}
}

func TestQuoteMakesAPathOneShellArgument(t *testing.T) {
	if got := quote("darwin", "/Users/Jo O'Neil/.jstack/repos/me/x"); got != `'/Users/Jo O'\''Neil/.jstack/repos/me/x'` {
		t.Fatalf("sh quoted = %s", got)
	}
	if got := quote("windows", `C:\Users\Jo O'Neil\.jstack\repos\me\x`); got != `'C:\Users\Jo O''Neil\.jstack\repos\me\x'` {
		t.Fatalf("powershell quoted = %s", got)
	}
}

func TestPullLineChangesIntoTheCloneAndSyncsFromTheRepoItself(t *testing.T) {
	if got := pullLine("linux", "me/x", "/home/jo/.jstack/repos/me/x"); got != "cd '/home/jo/.jstack/repos/me/x' && gh repo sync --source me/x" {
		t.Fatalf("sh line = %s", got)
	}
	if got := pullLine("windows", "me/x", `C:\Users\jo\.jstack\repos\me\x`); got != `Set-Location 'C:\Users\jo\.jstack\repos\me\x' -ErrorAction Stop; gh repo sync --source me/x` {
		t.Fatalf("powershell line = %s", got)
	}
}

func TestRepoQuestionIsAskedOnceAndRememberedWhenSkipped(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRoast("1.1.0")
	opts, out := options(t, home, shell, "")
	session, err := Start(opts)
	if err != nil {
		t.Fatal(err)
	}
	if session.SkillRepoAsked() {
		t.Fatal("the question counts as asked before the first run")
	}
	if err := guided(t, opts, script{}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "skill repos") {
		t.Fatalf("a skipped question printed a repos section:\n%s", out.String())
	}
	if got := read(t, filepath.Join(home, ".jstack", "config.json")); !strings.Contains(got, `"skill_repos_asked": true`) || strings.Contains(got, `"skill_repos"`+`:`) {
		t.Fatalf("config = %q", got)
	}
	if session, err = Start(opts); err != nil {
		t.Fatal(err)
	}
	if !session.SkillRepoAsked() {
		t.Fatal("asked again")
	}
}

func TestNoSkillRepoFlagRecordsTheAnswerHeadlessly(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRoast("1.1.0")
	opts, out := options(t, home, shell, "")
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "add --skill-repo owner/name to also install the skills from a repo of yours, or --no-skill-repo to say there is none")
	opts, out = options(t, home, shell, "")
	opts.Yes = true
	opts.NoSkillRepo = true
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(home, ".jstack", "config.json")); !strings.Contains(got, `"skill_repos_asked": true`) {
		t.Fatalf("config = %q", got)
	}
	opts, out = options(t, home, shell, "")
	session, err := Start(opts)
	if err != nil {
		t.Fatal(err)
	}
	if !session.SkillRepoAsked() {
		t.Fatal("asked after the answer was recorded")
	}
	if err := guided(t, opts, script{}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "--no-skill-repo") {
		t.Fatalf("hinted after the answer was recorded:\n%s", out.String())
	}
}

func TestRepoQuestionRejectsABadNameAndTakesAGoodOne(t *testing.T) {
	if _, err := RepoName("not a repo"); err == nil || !strings.Contains(err.Error(), `"not a repo" is not owner/name`) {
		t.Fatalf("err = %v", err)
	}
	home := homeWithClaude(t)
	shell := withRepo()
	opts, out := options(t, home, shell, "")
	opts.Overrides = map[string]string{"voice": "jstack"}
	if err := guided(t, opts, script{skillRepo: "https://github.com/me/work-skills.git"}); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "me/work-skills  ~/.jstack/repos/me/work-skills, cloned, 2 skills")
}

func TestRepoSkillsInstallBesideJstacksWithTheirSourceNamed(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRepo()
	opts, out := options(t, home, shell, "")
	opts.Yes = true
	opts.SkillRepos = []string{"https://github.com/me/work-skills"}
	opts.Overrides = map[string]string{"voice": "jstack"}
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(),
		"skill repos\n  me/work-skills  ~/.jstack/repos/me/work-skills, cloned, 2 skills\n  voice  kept from jstack, not installed from me/work-skills\n",
		"new      deploy (me/work-skills), how\n",
		"changed  voice\n",
		"local    mine (untouched)",
		"skills   2 installed, 1 updated in ~/.claude/skills",
	)
	if got := read(t, filepath.Join(home, ".claude", "skills", "deploy", "SKILL.md")); got != "deploy\n" {
		t.Fatalf("deploy = %q", got)
	}
	if got := read(t, filepath.Join(home, ".claude", "skills", "voice", "SKILL.md")); got != "voice v2\n" {
		t.Fatalf("voice = %q", got)
	}
	if got := read(t, filepath.Join(home, ".jstack", "config.json")); got != "{\n  \"harnesses\": [\n    \"claude\"\n  ],\n  \"harnesses_found\": [\n    \"claude\"\n  ],\n  \"skill_repos\": [\n    \"me/work-skills\"\n  ],\n  \"skill_repos_asked\": true,\n  \"skill_overrides\": {\n    \"voice\": \"jstack\"\n  }\n}\n" {
		t.Fatalf("config = %q", got)
	}
	clone := filepath.Join(home, ".jstack", "repos", "me", "work-skills")
	if got := shell.commands[0]; got != "gh repo clone me/work-skills "+quote(runtime.GOOS, clone) {
		t.Fatalf("first command = %q", got)
	}
}

func TestRerunPullsTheRepoAndUpdatesAChangedSkillWithABackup(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRepo()
	opts, _ := options(t, home, shell, "")
	opts.Yes = true
	opts.SkillRepos = []string{workSkills}
	opts.Overrides = map[string]string{"voice": "jstack"}
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	shell.repos[workSkills]["skills/deploy/SKILL.md"] = "deploy v2\n"
	shell.commands = nil
	opts, out := options(t, home, shell, "")
	opts.Yes = true
	opts.Now = func() time.Time { return time.Date(2026, 9, 3, 11, 0, 0, 0, time.UTC) }
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	clone := filepath.Join(home, ".jstack", "repos", "me", "work-skills")
	if got := shell.commands[0]; got != pullLine(runtime.GOOS, workSkills, clone) {
		t.Fatalf("first command = %q", got)
	}
	expectAll(t, out.String(),
		"me/work-skills  ~/.jstack/repos/me/work-skills, pulled, 2 skills",
		"voice  kept from jstack, not installed from me/work-skills",
		"changed  deploy (me/work-skills)\n",
		"skills   0 installed, 1 updated in ~/.claude/skills",
		"backup   ~/.jstack/backup/20260903-110000/claude/skills",
	)
	if got := read(t, filepath.Join(home, ".claude", "skills", "deploy", "SKILL.md")); got != "deploy v2\n" {
		t.Fatalf("deploy = %q", got)
	}
	if got := read(t, filepath.Join(home, ".jstack", "backup", "20260903-110000", "claude", "skills", "deploy", "SKILL.md")); got != "deploy\n" {
		t.Fatalf("backup = %q", got)
	}
}

func TestCollisionAsksAndRemembersThePick(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRepo()
	opts, out := options(t, home, shell, "")
	session, err := Start(opts)
	if err != nil {
		t.Fatal(err)
	}
	open, err := session.Gather(context.Background(), []string{workSkills})
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 1 || open[0].Name != "voice" || strings.Join(open[0].Sources, ",") != "jstack,me/work-skills" {
		t.Fatalf("open collisions = %+v", open)
	}
	session.Close()
	if err := guided(t, opts, script{skillRepo: workSkills, picks: map[string]string{"voice": workSkills}}); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(),
		"voice  overridden by me/work-skills, not installed from jstack",
		"changed  voice (me/work-skills)\n",
	)
	if got := read(t, filepath.Join(home, ".claude", "skills", "voice", "SKILL.md")); got != "voice, my way\n" {
		t.Fatalf("voice = %q", got)
	}
	if got := read(t, filepath.Join(home, ".jstack", "config.json")); !strings.Contains(got, "\"skill_overrides\": {\n    \"voice\": \"me/work-skills\"\n  }") {
		t.Fatalf("config = %q", got)
	}
	opts, out = options(t, home, shell, "")
	if session, err = Start(opts); err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if open, err = session.Gather(context.Background(), []string{workSkills}); err != nil || len(open) != 0 {
		t.Fatalf("asked again: open = %+v, err = %v", open, err)
	}
	if err := guided(t, opts, script{}); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "voice  overridden by me/work-skills, not installed from jstack", "same     3 skills")
}

func TestRenameChoiceStopsSetupWithTheHarnessesUntouched(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRepo()
	opts, _ := options(t, home, shell, "")
	err := guided(t, opts, script{skillRepo: workSkills, picks: map[string]string{"voice": Rename}})
	if err == nil || !strings.Contains(err.Error(), `rename the "voice" folder in me/work-skills`) {
		t.Fatalf("err = %v", err)
	}
	if read(t, filepath.Join(home, ".claude", "skills", "voice", "SKILL.md")) != "voice v1\n" || exists(filepath.Join(home, ".jstack", "config.json")) {
		t.Fatal("setup went on after the rename choice")
	}
}

func TestCollisionWithoutATerminalRefusesWithTheFlag(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRepo()
	for index, yes := range []bool{false, true} {
		opts, out := options(t, home, shell, "")
		opts.Yes = yes
		opts.SkillRepos = []string{workSkills}
		err := Run(context.Background(), opts)
		if err == nil || err.Error() != `[JSTACK-SKILL-COLLISION] skill "voice" is in jstack and me/work-skills, and there is no terminal to ask which one goes into the harnesses; rerun with --override voice=jstack to keep jstack's, --override voice=me/work-skills to use me/work-skills's, or rename the folder in me/work-skills` {
			t.Fatalf("yes=%v: err = %v", yes, err)
		}
		expectAll(t, out.String(), "me/work-skills  ~/.jstack/repos/me/work-skills, "+[]string{"cloned", "pulled"}[index]+", 2 skills")
		if read(t, filepath.Join(home, ".claude", "skills", "voice", "SKILL.md")) != "voice v1\n" {
			t.Fatal("a refused run changed a skill")
		}
	}
}

func TestOverrideFlagMustNameARealCollisionAndSource(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRepo()
	opts, _ := options(t, home, shell, "")
	opts.Yes = true
	opts.SkillRepos = []string{workSkills}
	opts.Overrides = map[string]string{"deploy": workSkills}
	err := Run(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), `[JSTACK-OVERRIDE] skill "deploy" is not in more than one source`) || !strings.Contains(err.Error(), "voice (jstack, me/work-skills)") {
		t.Fatalf("err = %v", err)
	}
	opts.Overrides = map[string]string{"voice": "me/other"}
	err = Run(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), `[JSTACK-OVERRIDE] skill "voice" is not in me/other; it is in jstack and me/work-skills; example: --override voice=me/work-skills`) {
		t.Fatalf("err = %v", err)
	}
}

func TestRerunLineCarriesTheRepoFlagsAndTheOverride(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRepo()
	opts, out := options(t, home, shell, "")
	opts.SkillRepos = []string{workSkills}
	opts.ForgetSkillRepos = []string{"me/old"}
	opts.Overrides = map[string]string{"voice": "jstack"}
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "jstack setup --harness claude --yes --skill-repo me/work-skills --forget-skill-repo me/old --override voice=jstack\n")
	if strings.Contains(out.String(), "add --skill-repo") {
		t.Fatalf("hinted the repo flag with a repo given:\n%s", out.String())
	}
	if exists(filepath.Join(home, ".claude", "skills", "deploy")) {
		t.Fatal("a plan-only run installed a skill")
	}
}

func TestUnreachableRepoIsReportedAndSetupCarriesOn(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRoast("1.1.0")
	opts, out := options(t, home, shell, "")
	opts.Yes = true
	opts.SkillRepos = []string{"me/private"}
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(),
		"me/private  ~/.jstack/repos/me/private, FAILED: `gh repo clone me/private "+quote(runtime.GOOS, filepath.Join(home, ".jstack", "repos", "me", "private"))+"` failed: exit status 1, GraphQL: Could not resolve to a Repository with the name 'me/private'. (repository); if the repo is private, check `gh auth status`; setup carries on without it",
		"skills   1 installed, 1 updated in ~/.claude/skills",
	)
	if got := read(t, filepath.Join(home, ".jstack", "config.json")); !strings.Contains(got, `"me/private"`) {
		t.Fatalf("a repo that failed once was forgotten: %q", got)
	}
}

func TestRepoWithoutASkillsFolderIsReported(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRepo()
	shell.repos["me/notes"] = map[string]string{"README.md": "no skills here\n"}
	opts, out := options(t, home, shell, "")
	opts.Yes = true
	opts.SkillRepos = []string{"me/notes"}
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "me/notes  ~/.jstack/repos/me/notes, FAILED: it has no skills/ folder; add one with a folder per skill, each with a SKILL.md, and push; setup carries on without it")
}

func TestForgettingARepoLeavesItsSkillsAsLocal(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRepo()
	opts, _ := options(t, home, shell, "")
	opts.Yes = true
	opts.SkillRepos = []string{workSkills}
	opts.Overrides = map[string]string{"voice": workSkills}
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	shell.commands = nil
	opts, out := options(t, home, shell, "")
	opts.Yes = true
	opts.ForgetSkillRepos = []string{workSkills}
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "changed  voice\n", "local    deploy, mine, roast (untouched)")
	if strings.Contains(out.String(), "skill repos") {
		t.Fatalf("a forgotten repo was still listed:\n%s", out.String())
	}
	for _, command := range shell.commands {
		if strings.Contains(command, "gh repo ") {
			t.Fatalf("a forgotten repo was pulled: %q", command)
		}
	}
	if got := read(t, filepath.Join(home, ".jstack", "config.json")); strings.Contains(got, "work-skills") {
		t.Fatalf("config still names the repo: %q", got)
	}
	if !exists(filepath.Join(home, ".claude", "skills", "deploy", "SKILL.md")) {
		t.Fatal("the repo's skill was removed from the harness")
	}
}

func TestTwoReposCollidingAskBetweenThem(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRepo()
	shell.repos["me/more"] = map[string]string{"skills/deploy/SKILL.md": "deploy, the other way\n"}
	opts, out := options(t, home, shell, "")
	opts.Yes = true
	opts.SkillRepos = []string{workSkills, "me/more"}
	opts.Overrides = map[string]string{"voice": "jstack", "deploy": "me/more"}
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(),
		"deploy  overridden by me/more, not installed from me/work-skills",
		"voice  kept from jstack, not installed from me/work-skills",
		"new      deploy (me/more), how\n",
	)
	if got := read(t, filepath.Join(home, ".claude", "skills", "deploy", "SKILL.md")); got != "deploy, the other way\n" {
		t.Fatalf("deploy = %q", got)
	}
	if entries, _ := os.ReadDir(filepath.Join(home, ".jstack", "repos", "me")); len(entries) != 2 {
		t.Fatalf("clones = %v", entries)
	}
}

func TestFailedPullKeepsTheLastCopyAndTheSavedPick(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRepo()
	opts, _ := options(t, home, shell, "")
	opts.Yes = true
	opts.SkillRepos = []string{workSkills}
	opts.Overrides = map[string]string{"voice": workSkills}
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	clone := filepath.Join(home, ".jstack", "repos", "me", "work-skills")
	shell.failing = map[string]bool{pullLine(runtime.GOOS, workSkills, clone): true}
	opts, out := options(t, home, shell, "")
	opts.Yes = true
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(),
		"me/work-skills  ~/.jstack/repos/me/work-skills, not pulled, using the copy from the last run: `"+pullLine(runtime.GOOS, workSkills, clone)+"` failed: exit status 1; if the repo is private, check `gh auth status`, 2 skills",
		"voice  overridden by me/work-skills, not installed from jstack",
		"changed  -\n",
	)
	if got := read(t, filepath.Join(home, ".claude", "skills", "voice", "SKILL.md")); got != "voice, my way\n" {
		t.Fatalf("voice reverted to jstack's: %q", got)
	}
	if got := read(t, filepath.Join(home, ".jstack", "config.json")); !strings.Contains(got, `"voice": "me/work-skills"`) {
		t.Fatalf("the pick was lost: %q", got)
	}
}

func TestLostCloneAndDeadNetworkKeepTheSkillAsInstalled(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRepo()
	opts, _ := options(t, home, shell, "")
	opts.Yes = true
	opts.SkillRepos = []string{workSkills}
	opts.Overrides = map[string]string{"voice": workSkills}
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(home, ".jstack", "repos")); err != nil {
		t.Fatal(err)
	}
	delete(shell.repos, workSkills)
	opts, out := options(t, home, shell, "")
	opts.Yes = true
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(),
		"FAILED: `gh repo clone me/work-skills",
		"voice  left as it is in each harness, me/work-skills couldn't be reached this run; a harness without it gets it once the repo is reached\n",
		"changed  -\n",
		"local    deploy, mine, roast, voice (untouched)",
	)
	if got := read(t, filepath.Join(home, ".claude", "skills", "voice", "SKILL.md")); got != "voice, my way\n" {
		t.Fatalf("voice reverted to jstack's: %q", got)
	}
	if got := read(t, filepath.Join(home, ".jstack", "config.json")); !strings.Contains(got, `"voice": "me/work-skills"`) {
		t.Fatalf("the pick was lost: %q", got)
	}
}

func TestSavedPicksArePrunedOnlyOnARunThatReachedEveryRepo(t *testing.T) {
	saved := map[string]string{"voice": workSkills, "how": "jstack", "deploy": "me/down"}
	got := rememberOverrides(saved, []skillRepo{{name: "me/down", failure: "no network"}}, map[string]string{"why": workSkills})
	if len(got) != 4 || got["deploy"] != "me/down" || got["how"] != "jstack" || got["why"] != workSkills {
		t.Fatalf("remembered with a repo down = %v", got)
	}
	got = rememberOverrides(saved, nil, map[string]string{"voice": "jstack"})
	if len(got) != 1 || got["voice"] != "jstack" {
		t.Fatalf("remembered with every repo reached = %v", got)
	}
}

func TestSkillsFolderThatIsASymlinkOutOfTheCloneIsRefused(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need a privilege on windows")
	}
	home := homeWithClaude(t)
	shell := withRepo()
	shell.repos["me/sneaky"] = map[string]string{"README.md": "skills is a link\n"}
	opts, _ := options(t, home, shell, "")
	opts.Yes = true
	opts.SkillRepos = []string{"me/sneaky"}
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	clone := filepath.Join(home, ".jstack", "repos", "me", "sneaky")
	if err := os.Symlink(filepath.Join(home, ".claude", "skills"), filepath.Join(clone, "skills")); err != nil {
		t.Fatal(err)
	}
	opts, out := options(t, home, shell, "")
	opts.Yes = true
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "me/sneaky  ~/.jstack/repos/me/sneaky, FAILED: cannot open its skills/ folder: ", "not a symlink out of it; setup carries on without it")
	if strings.Contains(out.String(), "mine (me/sneaky)") {
		t.Fatalf("the harness's own skills were taken as the repo's:\n%s", out.String())
	}
}

func TestRepoSkillWithACapitalLetterIsLeftOut(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRepo()
	shell.repos[workSkills]["skills/Notes/SKILL.md"] = "Notes, capitalized\n"
	opts, out := options(t, home, shell, "")
	opts.Yes = true
	opts.SkillRepos = []string{workSkills}
	opts.Overrides = map[string]string{"voice": "jstack"}
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(),
		"me/work-skills  ~/.jstack/repos/me/work-skills, cloned, 2 skills\n  Notes  not a lowercase name, the copy in me/work-skills is left out; rename the folder\n",
		"new      deploy (me/work-skills), how\n",
	)
	if exists(filepath.Join(home, ".claude", "skills", "Notes")) {
		t.Fatal("the capitalized folder was installed")
	}
}

func TestToolSkillFoldersInARepoAreLeftOut(t *testing.T) {
	home := homeWithClaude(t)
	shell := withRepo()
	shell.repos[workSkills]["skills/roast/SKILL.md"] = "roast, my copy\n"
	shell.repos[workSkills]["skills/roast/refs/notes.md"] = "notes\n"
	opts, out := options(t, home, shell, "")
	opts.Yes = true
	opts.SkillRepos = []string{workSkills}
	opts.Overrides = map[string]string{"voice": "jstack"}
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(),
		"me/work-skills  ~/.jstack/repos/me/work-skills, cloned, 2 skills\n  roast  installed by the roast tool itself, the copy in me/work-skills is left out\n",
		"new      deploy (me/work-skills), how\n",
		"ok roast 1.1.0, skill installed via roast install-skill",
	)
	if exists(filepath.Join(home, ".claude", "skills", "roast", "refs")) {
		t.Fatal("the repo's roast folder was installed")
	}
}

func TestRepoSymlinkPointingOutsideTheCloneIsRefused(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need a privilege on windows")
	}
	home := homeWithClaude(t)
	shell := withRepo()
	opts, _ := options(t, home, shell, "")
	opts.Yes = true
	opts.SkillRepos = []string{workSkills}
	opts.Overrides = map[string]string{"voice": "jstack"}
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(home, ".aws", "credentials"), "aws_secret_access_key = hunter2\n")
	clone := filepath.Join(home, ".jstack", "repos", "me", "work-skills")
	if err := os.Symlink(filepath.Join(home, ".aws", "credentials"), filepath.Join(clone, "skills", "deploy", "creds")); err != nil {
		t.Fatal(err)
	}
	opts, _ = options(t, home, shell, "")
	opts.Yes = true
	err := Run(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "JSTACK-SKILLS-SOURCE") || !strings.Contains(err.Error(), `skill "deploy" from me/work-skills`) || !strings.Contains(err.Error(), "symlink pointing outside") {
		t.Fatalf("err = %v", err)
	}
	if exists(filepath.Join(home, ".claude", "skills", "deploy", "creds")) {
		t.Fatal("the credentials file was copied into the harness")
	}
}
