package setup

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// homeWithRepos is a home with ~/code holding four checkouts: alpha declares
// its tracker in AGENTS.md, bravo has an AGENTS.md with no line, charlie has
// no instructions file at all, and delta declares it in CLAUDE.md, the only
// file it keeps. A notes folder without .git and a stray file are there to
// be skipped.
func homeWithRepos(t *testing.T) string {
	t.Helper()
	home := homeWithClaude(t)
	code := filepath.Join(home, "code")
	for _, name := range []string{"alpha", "bravo", "charlie", "delta"} {
		if err := os.MkdirAll(filepath.Join(code, name, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write(t, filepath.Join(code, "alpha", "AGENTS.md"), "# Alpha\n\nTracker: markdown tasks/\n\nThe rest.\n")
	write(t, filepath.Join(code, "bravo", "AGENTS.md"), "# Bravo\n\nSome text.\n")
	write(t, filepath.Join(code, "delta", "CLAUDE.md"), "Tracker: jira SR\n")
	write(t, filepath.Join(code, "notes", "README.md"), "not a checkout\n")
	write(t, filepath.Join(code, "stray.txt"), "not a folder\n")
	return home
}

// savedRepos is the config a run that already answered both questions
// leaves behind, so a test can start at the tracker questions.
func savedRepos(t *testing.T, home string) {
	t.Helper()
	write(t, filepath.Join(home, ".jstack", "config.json"), `{"harnesses":["claude"],"skill_repos_asked":true,"repos_dirs":["`+strings.ReplaceAll(filepath.Join(home, "code"), `\`, `\\`)+`"],"repos_dirs_asked":true}`)
}

func in(home, repo, command string) string {
	return inRepo(runtime.GOOS, filepath.Join(home, "code", repo), command)
}

// answering is a tracker script: the same line for every undeclared repo,
// the PR only for the repos named, where it's offered.
func answering(line string, prFor ...string) func([]TrackerQuestion) []TrackerAnswer {
	return func(questions []TrackerQuestion) []TrackerAnswer {
		var answers []TrackerAnswer
		for _, question := range questions {
			openPR := false
			for _, name := range prFor {
				openPR = openPR || name == question.Repo
			}
			answers = append(answers, TrackerAnswer{Repo: question.Repo, Line: line, OpenPR: openPR && question.PROffer})
		}
		return answers
	}
}

// offers is which repos the questions offer the PR for, as "bravo:yes charlie:no".
func offers(questions []TrackerQuestion) string {
	var parts []string
	for _, question := range questions {
		state := "no"
		if question.PROffer {
			state = "yes"
		}
		parts = append(parts, question.Repo+":"+state)
	}
	return strings.Join(parts, " ")
}

func TestWithTrackerLineGoesAfterTheFirstHeading(t *testing.T) {
	line := "Tracker: linear SR"
	for _, tc := range []struct{ name, content, want string }{
		{"heading then blank", "# Title\n\nBody.\n", "# Title\n\nTracker: linear SR\n\nBody.\n"},
		{"heading then text", "# Title\nBody.\n", "# Title\n\nTracker: linear SR\n\nBody.\n"},
		{"heading only, no newline", "# Title", "# Title\n\nTracker: linear SR\n"},
		{"heading that is not on line 1", "Intro.\n\n## Later\n", "Tracker: linear SR\n\nIntro.\n\n## Later\n"},
		{"no heading", "Body.\n", "Tracker: linear SR\n\nBody.\n"},
		{"empty", "", "Tracker: linear SR\n"},
	} {
		if got := withTrackerLine(tc.content, line); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestScanReadsAgentsMdThenClaudeMdAndSkipsWhatIsNotACheckout(t *testing.T) {
	home := homeWithRepos(t)
	repos := scanRepos([]string{filepath.Join(home, "code")})
	got := make([]string, 0, len(repos))
	for _, repo := range repos {
		got = append(got, repo.name+"="+repo.line+"@"+filepath.Base(repo.file))
	}
	if strings.Join(got, ";") != "alpha=Tracker: markdown tasks/@AGENTS.md;bravo=@AGENTS.md;charlie=@AGENTS.md;delta=Tracker: jira SR@CLAUDE.md" {
		t.Fatalf("repos = %v", got)
	}
}

func TestLineGoesIntoClaudeMdWhenThatIsTheOnlyFile(t *testing.T) {
	home := homeWithRepos(t)
	dir := filepath.Join(home, "code", "echo")
	write(t, filepath.Join(dir, ".git", "HEAD"), "ref: refs/heads/main\n")
	write(t, filepath.Join(dir, "CLAUDE.md"), "# Echo\n")
	file, line := readTracker(dir)
	if filepath.Base(file) != "CLAUDE.md" || line != "" {
		t.Fatalf("file = %s, line = %q", file, line)
	}
	shell := withRoast("1.1.0")
	shell.failing = map[string]bool{in(home, "echo", "git remote get-url --push origin"): true}
	opts, out := options(t, home, shell, "")
	if err := declareTracker(context.Background(), opts, trackerRepo{dir: dir, name: "echo", file: file}, "Tracker: github-issues", false); err != nil {
		t.Fatal(err)
	}
	if got := read(t, file); got != "# Echo\n\nTracker: github-issues\n" {
		t.Fatalf("CLAUDE.md = %q", got)
	}
	expectAll(t, out.String(), `echo  wrote "Tracker: github-issues" to ~/code/echo/CLAUDE.md`, "echo  has no origin remote, so the line is left uncommitted")
}

func TestReposFolderIsGuessedAskedOnceAndRemembered(t *testing.T) {
	home := homeWithRepos(t)
	shell := withRoast("1.1.0")
	opts, out := options(t, home, shell, "")
	session, err := Start(opts)
	if err != nil {
		t.Fatal(err)
	}
	if session.ReposDirsAsked() || strings.Join(session.ReposDirGuesses(), ",") != filepath.Join(home, "code") {
		t.Fatalf("asked = %v, guesses = %v", session.ReposDirsAsked(), session.ReposDirGuesses())
	}
	if err := guided(t, opts, script{}); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(),
		"repos ~/code\n  alpha    Tracker: markdown tasks/\n  bravo    not declared\n  charlie  not declared\n  delta    Tracker: jira SR\n",
		"\ntrackers\n  bravo    skipped this run\n  charlie  skipped this run\n",
		"bravo  skipped\n  charlie  skipped",
	)
	if got := read(t, filepath.Join(home, ".jstack", "config.json")); !strings.Contains(got, `"repos_dirs": [`) || !strings.Contains(got, `"repos_dirs_asked": true`) {
		t.Fatalf("config = %q", got)
	}
	if read(t, filepath.Join(home, "code", "bravo", "AGENTS.md")) != "# Bravo\n\nSome text.\n" || exists(filepath.Join(home, "code", "charlie", "AGENTS.md")) {
		t.Fatal("a skipped repo changed")
	}
	opts, out = options(t, home, shell, "")
	if session, err = Start(opts); err != nil {
		t.Fatal(err)
	}
	if !session.ReposDirsAsked() {
		t.Fatal("asked for the folder again")
	}
	if err := guided(t, opts, script{}); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "repos ~/code\n  alpha    Tracker: markdown tasks/\n  bravo    not declared\n", "bravo  skipped")
}

func TestTypedReposFolderMustExist(t *testing.T) {
	home := homeWithRepos(t)
	if err := os.Rename(filepath.Join(home, "code"), filepath.Join(home, "work")); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseReposDirs("nowhere", home); err == nil || !strings.Contains(err.Error(), `[JSTACK-REPOS-DIR] "nowhere" is not a folder; type the path of a folder that exists, such as `+filepath.Join(home, "code")) {
		t.Fatalf("err = %v", err)
	}
	dirs, err := ParseReposDirs("~/work", home)
	if err != nil || strings.Join(dirs, ",") != filepath.Join(home, "work") {
		t.Fatalf("dirs = %v, err = %v", dirs, err)
	}
	opts, out := options(t, home, withRoast("1.1.0"), "")
	if err := guided(t, opts, script{reposDirs: dirs}); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "repos ~/work\n  alpha    Tracker: markdown tasks/\n")
}

func TestReposDirFlagSettlesTheQuestionWithoutATerminal(t *testing.T) {
	home := homeWithRepos(t)
	opts, out := options(t, home, withRoast("1.1.0"), "")
	opts.Yes = true
	opts.ReposDirs = []string{filepath.Join(home, "code")}
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "repos ~/code\n  alpha    Tracker: markdown tasks/\n  bravo    not declared\n", "\nrepos\n  2 not declared: bravo, charlie; rerun with a terminal to name the tracker of each one\n")
	if got := read(t, filepath.Join(home, ".jstack", "config.json")); !strings.Contains(got, `"repos_dirs_asked": true`) {
		t.Fatalf("config = %q", got)
	}
	if read(t, filepath.Join(home, "code", "bravo", "AGENTS.md")) != "# Bravo\n\nSome text.\n" {
		t.Fatal("bravo changed without a terminal")
	}
}

func TestNoTerminalReportsTheReposAndChangesNothing(t *testing.T) {
	home := homeWithRepos(t)
	savedRepos(t, home)
	shell := withRoast("1.1.0")
	opts, out := options(t, home, shell, "")
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "repos ~/code\n  alpha    Tracker: markdown tasks/\n  bravo    not declared\n  charlie  not declared\n  delta    Tracker: jira SR\n", "2 repo(s) declare no tracker: bravo, charlie; rerun with a terminal to name each one")
	if read(t, filepath.Join(home, "code", "bravo", "AGENTS.md")) != "# Bravo\n\nSome text.\n" || exists(filepath.Join(home, "code", "charlie", "AGENTS.md")) {
		t.Fatal("a repo changed without a terminal")
	}
	if strings.Join(shell.commands, ";") != "check-git;check-roast;version-roast" {
		t.Fatalf("commands = %v", shell.commands)
	}
}

func TestNoFolderNamedHintsTheFlagWithTheGuesses(t *testing.T) {
	home := homeWithRepos(t)
	opts, out := options(t, home, withRoast("1.1.0"), "")
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "\nrepos\n  none named, so the trackers are not checked; add --repos-dir <folder> to name where your repos live, found ~/code\n")
	opts, out = options(t, home, withRoast("1.1.0"), "")
	opts.ReposDirs = []string{filepath.Join(home, "code")}
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "jstack setup --harness claude --yes --repos-dir "+quote(runtime.GOOS, filepath.Join(home, "code")))
}

func TestTrackerAnswersWriteTheLineAndOpenThePRThroughGh(t *testing.T) {
	home := homeWithRepos(t)
	savedRepos(t, home)
	shell := withRoast("1.1.0")
	shell.versions[in(home, "bravo", "git rev-parse --abbrev-ref HEAD")] = "main"
	shell.failing = map[string]bool{in(home, "charlie", "git remote get-url --push origin"): true}
	opts, out := options(t, home, shell, "")
	err := guided(t, opts, script{trackers: func(questions []TrackerQuestion) []TrackerAnswer {
		if got := offers(questions); got != "bravo:yes charlie:no" {
			t.Fatalf("offers = %q", got)
		}
		if questions[0].File != "~/code/bravo/AGENTS.md" || questions[1].File != "~/code/charlie/AGENTS.md" {
			t.Fatalf("files = %+v", questions)
		}
		linear, markdown := Backends()[2], Backends()[0]
		return []TrackerAnswer{
			{Repo: "bravo", Line: TrackerLine(linear, "SR"), OpenPR: true},
			{Repo: "charlie", Line: TrackerLine(markdown, markdown.Default)},
		}
	}})
	if err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(),
		"\ntrackers\n  bravo    would get \"Tracker: linear SR\" and a PR on branch tracker-line\n  charlie  would get \"Tracker: markdown tasks/\", left uncommitted\n",
		`bravo  wrote "Tracker: linear SR" to ~/code/bravo/AGENTS.md`,
		"bravo  PR opened, back on main",
		`charlie  wrote "Tracker: markdown tasks/" to ~/code/charlie/AGENTS.md`,
		"charlie  has no origin remote, so the line is left uncommitted; commit it yourself",
	)
	if got := read(t, filepath.Join(home, "code", "bravo", "AGENTS.md")); got != "# Bravo\n\nTracker: linear SR\n\nSome text.\n" {
		t.Fatalf("bravo AGENTS.md = %q", got)
	}
	if got := read(t, filepath.Join(home, "code", "charlie", "AGENTS.md")); got != "Tracker: markdown tasks/\n" {
		t.Fatalf("charlie AGENTS.md = %q", got)
	}
	body := quote(runtime.GOOS, trackerPRBody)
	reads := func(repo string) []string {
		return []string{
			in(home, repo, "git status --porcelain"),
			in(home, repo, "git remote get-url --push origin"),
			in(home, repo, "git rev-parse --abbrev-ref HEAD"),
			in(home, repo, "git symbolic-ref --short refs/remotes/origin/HEAD"),
			in(home, repo, "git rev-parse HEAD "+quote(runtime.GOOS, "refs/remotes/origin/main")),
		}
	}
	var expected, got []string
	for _, command := range shell.commands {
		if strings.Contains(command, filepath.Join("code", "bravo")) || strings.Contains(command, filepath.Join("code", "charlie")) {
			got = append(got, command)
		}
	}
	expected = append(expected, reads("bravo")...)
	expected = append(expected, reads("charlie")...)
	expected = append(expected, reads("bravo")...)
	expected = append(expected,
		in(home, "bravo", "gh repo view --json name "+quote(runtime.GOOS, "git@github.com:me/bravo.git")),
		in(home, "bravo", "git checkout -b tracker-line"),
		in(home, "bravo", "git add "+quote(runtime.GOOS, "AGENTS.md")),
		in(home, "bravo", "git commit -m "+quote(runtime.GOOS, "docs: name the tracker")),
		in(home, "bravo", "git -c credential.helper="+quote(runtime.GOOS, "!gh auth git-credential")+" push -u origin tracker-line"),
		in(home, "bravo", "gh pr create --title "+quote(runtime.GOOS, "docs: name the tracker")+" --body "+body),
		in(home, "bravo", "git checkout "+quote(runtime.GOOS, "main")),
	)
	expected = append(expected, reads("charlie")...)
	if strings.Join(got, "\n") != strings.Join(expected, "\n") {
		t.Fatalf("commands:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(expected, "\n"))
	}
	if !strings.Contains(trackerPRBody, "Problem:") || !strings.Contains(trackerPRBody, "Fix:") || !strings.Contains(trackerPRBody, "Done when:\n- ") {
		t.Fatalf("PR body is not in the ticket shape:\n%s", trackerPRBody)
	}
}

func TestSkipLeavesTheRepoAsItIsForThisRun(t *testing.T) {
	home := homeWithRepos(t)
	savedRepos(t, home)
	shell := withRoast("1.1.0")
	opts, out := options(t, home, shell, "")
	err := guided(t, opts, script{trackers: func([]TrackerQuestion) []TrackerAnswer {
		return []TrackerAnswer{{Repo: "bravo", Skip: true}, {Repo: "charlie", Line: TrackerLine(Backends()[3], "SR")}}
	}})
	if err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "bravo    skipped this run", "bravo  skipped", `charlie  wrote "Tracker: jira SR"`)
	if read(t, filepath.Join(home, "code", "bravo", "AGENTS.md")) != "# Bravo\n\nSome text.\n" {
		t.Fatal("a skipped repo changed")
	}
	for _, command := range shell.commands {
		if strings.Contains(command, "bravo") && (strings.Contains(command, "checkout") || strings.Contains(command, "git add") || strings.Contains(command, "commit") || strings.Contains(command, "push -u")) {
			t.Fatalf("a skipped repo ran %q", command)
		}
	}
}

func TestDirtyTreeWritesTheLineAndSkipsThePROffer(t *testing.T) {
	home := homeWithRepos(t)
	savedRepos(t, home)
	shell := withRoast("1.1.0")
	shell.versions[in(home, "bravo", "git status --porcelain")] = " M README.md"
	opts, out := options(t, home, shell, "")
	err := guided(t, opts, script{trackers: func(questions []TrackerQuestion) []TrackerAnswer {
		if got := offers(questions); got != "bravo:no charlie:yes" {
			t.Fatalf("offered a PR on a dirty tree: %q", got)
		}
		return answering("Tracker: github-issues")(questions)
	}})
	if err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), `bravo  wrote "Tracker: github-issues"`, "bravo  has other uncommitted changes, so the line is left uncommitted with them", "charlie  line left uncommitted; commit it yourself")
}

func TestFailedPushGoesBackToTheBranchAndIsReportedAtTheEnd(t *testing.T) {
	home := homeWithRepos(t)
	savedRepos(t, home)
	shell := withRoast("1.1.0")
	shell.versions[in(home, "bravo", "git rev-parse --abbrev-ref HEAD")] = "main"
	shell.failing = map[string]bool{in(home, "bravo", "git -c credential.helper="+quote(runtime.GOOS, "!gh auth git-credential")+" push -u origin tracker-line"): true}
	opts, out := options(t, home, shell, "")
	err := guided(t, opts, script{trackers: answering("Tracker: github-issues", "bravo")})
	if err == nil || !strings.Contains(err.Error(), "[JSTACK-TRACKERS] the harnesses are done, but 1 repo step(s) failed") || !strings.Contains(err.Error(), "push -u origin tracker-line` failed") {
		t.Fatalf("err = %v", err)
	}
	expectAll(t, out.String(), "push -u origin tracker-line` failed: exit status 1; finish the PR by hand from", `charlie  wrote "Tracker: github-issues"`)
	commands := strings.Join(shell.commands, "\n")
	if strings.Contains(commands, "gh pr create") || !strings.Contains(commands, in(home, "bravo", "git checkout "+quote(runtime.GOOS, "main"))) {
		t.Fatalf("commands:\n%s", commands)
	}
}

func TestEveryRepoDeclaredAsksNothing(t *testing.T) {
	home := homeWithRepos(t)
	savedRepos(t, home)
	write(t, filepath.Join(home, "code", "bravo", "AGENTS.md"), "Tracker: linear SR\n")
	write(t, filepath.Join(home, "code", "charlie", "CLAUDE.md"), "# Charlie\n\nTracker: github-issues\n")
	opts, out := options(t, home, withRoast("1.1.0"), "")
	err := guided(t, opts, script{trackers: func(questions []TrackerQuestion) []TrackerAnswer {
		if len(questions) != 0 {
			t.Fatalf("asked with nothing undeclared: %+v", questions)
		}
		return nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "\nrepos\n  every repo names its tracker\n")
}

func TestRepoWhosePRIsStillOpenIsReportedAndNotAskedAgain(t *testing.T) {
	home := homeWithRepos(t)
	savedRepos(t, home)
	write(t, filepath.Join(home, "code", "bravo", ".git", "refs", "heads", "tracker-line"), "0123456789abcdef0123456789abcdef01234567\n")
	opts, out := options(t, home, withRoast("1.1.0"), "")
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "  bravo    not declared, the line waits on branch tracker-line until its PR merges; delete the branch to be asked again\n  charlie  not declared\n", "1 repo(s) declare no tracker: charlie; rerun with a terminal")
	opts, out = options(t, home, withRoast("1.1.0"), "")
	err := guided(t, opts, script{trackers: func(questions []TrackerQuestion) []TrackerAnswer {
		if got := offers(questions); got != "charlie:yes" {
			t.Fatalf("asked about a repo whose PR is open: %q", got)
		}
		return []TrackerAnswer{{Repo: "charlie", Skip: true}}
	}})
	if err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "charlie  skipped")
}

func TestReposDirFlagIsResolvedFromTheWorkingDirectoryAndMustExist(t *testing.T) {
	home := homeWithRepos(t)
	t.Chdir(home)
	opts, _ := options(t, home, withRoast("1.1.0"), "")
	opts.Yes = true
	opts.ReposDirs = []string{"code"}
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(home, ".jstack", "config.json")); !strings.Contains(got, strings.ReplaceAll(filepath.Join(home, "code"), `\`, `\\`)) || strings.Contains(got, `"code"`) {
		t.Fatalf("config = %q", got)
	}
	opts, _ = options(t, home, withRoast("1.1.0"), "")
	opts.ReposDirs = []string{"nowhere"}
	err := Run(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), `[JSTACK-REPOS-DIR] "nowhere" is not a folder`) {
		t.Fatalf("err = %v", err)
	}
}

func TestLinkedAgentsMdIsWrittenThroughAndItsTargetStaged(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("making a symlink on Windows needs a privilege the test runner may not have")
	}
	home := homeWithRepos(t)
	savedRepos(t, home)
	bravo := filepath.Join(home, "code", "bravo")
	write(t, filepath.Join(bravo, "CLAUDE.md"), "# Bravo\n\nSome text.\n")
	if err := os.Remove(filepath.Join(bravo, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("CLAUDE.md", filepath.Join(bravo, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	shell := withRoast("1.1.0")
	shell.versions[in(home, "bravo", "git rev-parse --abbrev-ref HEAD")] = "main"
	opts, out := options(t, home, shell, "")
	err := guided(t, opts, script{trackers: func([]TrackerQuestion) []TrackerAnswer {
		return []TrackerAnswer{{Repo: "bravo", Line: "Tracker: github-issues", OpenPR: true}, {Repo: "charlie", Skip: true}}
	}})
	if err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), `bravo  wrote "Tracker: github-issues" to ~/code/bravo/CLAUDE.md`, "bravo  PR opened, back on main")
	if got := read(t, filepath.Join(bravo, "CLAUDE.md")); got != "# Bravo\n\nTracker: github-issues\n\nSome text.\n" {
		t.Fatalf("CLAUDE.md = %q", got)
	}
	if info, err := os.Lstat(filepath.Join(bravo, "AGENTS.md")); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("AGENTS.md is no longer a link")
	}
	if !strings.Contains(strings.Join(shell.commands, "\n"), in(home, "bravo", "git add "+quote(runtime.GOOS, "CLAUDE.md"))) {
		t.Fatalf("commands:\n%s", strings.Join(shell.commands, "\n"))
	}
}

func TestAgentsMdLinkedToOutsideTheRepoIsLeftAlone(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("making a symlink on Windows needs a privilege the test runner may not have")
	}
	home := homeWithRepos(t)
	savedRepos(t, home)
	write(t, filepath.Join(home, "dotfiles", "AGENTS.md"), "# Shared\n")
	if err := os.Symlink(filepath.Join(home, "dotfiles", "AGENTS.md"), filepath.Join(home, "code", "charlie", "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	opts, out := options(t, home, withRoast("1.1.0"), "")
	err := guided(t, opts, script{trackers: func(questions []TrackerQuestion) []TrackerAnswer {
		if got := offers(questions); got != "bravo:yes" {
			t.Fatalf("asked about a repo whose file links outside it: %q", got)
		}
		return []TrackerAnswer{{Repo: "bravo", Skip: true}}
	}})
	if err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "  charlie  not declared, charlie's instructions file links to outside the repo, so setup leaves it alone\n", "bravo  skipped")
	if got := read(t, filepath.Join(home, "dotfiles", "AGENTS.md")); got != "# Shared\n" {
		t.Fatalf("the shared file changed: %q", got)
	}
}

func TestPRIsOfferedOnlyFromTheDefaultBranch(t *testing.T) {
	home := homeWithRepos(t)
	savedRepos(t, home)
	shell := withRoast("1.1.0")
	shell.versions[in(home, "bravo", "git rev-parse --abbrev-ref HEAD")] = "feature"
	shell.versions[in(home, "bravo", "git symbolic-ref --short refs/remotes/origin/HEAD")] = "origin/main"
	shell.failing = map[string]bool{in(home, "charlie", "git symbolic-ref --short refs/remotes/origin/HEAD"): true}
	opts, out := options(t, home, shell, "")
	err := guided(t, opts, script{trackers: func(questions []TrackerQuestion) []TrackerAnswer {
		if got := offers(questions); got != "bravo:no charlie:no" {
			t.Fatalf("offered a PR off the default branch: %q", got)
		}
		return answering("Tracker: github-issues", "bravo")(questions)
	}})
	if err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "bravo  is on branch feature, not main, so the line is left uncommitted; commit it yourself", "charlie  has no origin/HEAD, so setup can't tell its default branch and the line is left uncommitted; run `git remote set-head origin -a` there, then rerun")
	if strings.Contains(strings.Join(shell.commands, "\n"), "checkout -b") {
		t.Fatalf("opened a PR off the default branch:\n%s", out.String())
	}
}

func TestPendingBranchIsSeenInPackedRefsAndFromALinkedWorktree(t *testing.T) {
	home := homeWithRepos(t)
	savedRepos(t, home)
	write(t, filepath.Join(home, "code", "bravo", ".git", "packed-refs"), "# pack-refs with: peeled fully-peeled sorted \n0123456789abcdef0123456789abcdef01234567 refs/heads/main\n89abcdef0123456789abcdef0123456789abcdef refs/heads/tracker-line\n")
	main := filepath.Join(home, "elsewhere", "charlie-main")
	write(t, filepath.Join(main, ".git", "refs", "heads", "tracker-line"), "0123456789abcdef0123456789abcdef01234567\n")
	write(t, filepath.Join(main, ".git", "worktrees", "charlie", "commondir"), "../..\n")
	if err := os.Remove(filepath.Join(home, "code", "charlie", ".git")); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(home, "code", "charlie", ".git"), "gitdir: "+filepath.Join(main, ".git", "worktrees", "charlie")+"\n")
	opts, out := options(t, home, withRoast("1.1.0"), "")
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "  bravo    not declared, the line waits on branch tracker-line until its PR merges; delete the branch to be asked again\n  charlie  not declared, the line waits on branch tracker-line until its PR merges; delete the branch to be asked again\n")
	if strings.Contains(out.String(), "declare no tracker") {
		t.Fatalf("a waiting repo counted as undeclared:\n%s", out.String())
	}
}

func TestPRIsOfferedOnlyWhenTheDefaultBranchIsAtItsRemote(t *testing.T) {
	home := homeWithRepos(t)
	savedRepos(t, home)
	shell := withRoast("1.1.0")
	shell.versions[in(home, "bravo", "git rev-parse HEAD "+quote(runtime.GOOS, "refs/remotes/origin/main"))] = "1111111111111111111111111111111111111111\n2222222222222222222222222222222222222222"
	opts, out := options(t, home, shell, "")
	err := guided(t, opts, script{trackers: func(questions []TrackerQuestion) []TrackerAnswer {
		if got := offers(questions); got != "bravo:no charlie:yes" {
			t.Fatalf("offered a PR with local commits ahead: %q", got)
		}
		return answering("Tracker: github-issues")(questions)
	}})
	if err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "bravo  is on main but not at origin/main, so the line is left uncommitted; push or pull first, then rerun", "charlie  line left uncommitted; commit it yourself")
}

func TestLinkChainEndingOutsideTheRepoIsLeftAlone(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("making a symlink on Windows needs a privilege the test runner may not have")
	}
	home := homeWithRepos(t)
	savedRepos(t, home)
	charlie := filepath.Join(home, "code", "charlie")
	write(t, filepath.Join(home, "dotfiles", "AGENTS.md"), "# Shared\n")
	if err := os.Symlink(filepath.Join(home, "dotfiles", "AGENTS.md"), filepath.Join(charlie, "link.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("link.md", filepath.Join(charlie, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	opts, out := options(t, home, withRoast("1.1.0"), "")
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "  charlie  not declared, charlie's instructions file links to outside the repo, so setup leaves it alone\n")
}

func TestFailedCommitPutsTheBranchBackAndDeletesTheEmptyOne(t *testing.T) {
	home := homeWithRepos(t)
	savedRepos(t, home)
	shell := withRoast("1.1.0")
	shell.failing = map[string]bool{in(home, "bravo", "git commit -m "+quote(runtime.GOOS, "docs: name the tracker")): true}
	opts, _ := options(t, home, shell, "")
	err := guided(t, opts, script{trackers: answering("Tracker: github-issues", "bravo")})
	if err == nil || !strings.Contains(err.Error(), "the line is written and uncommitted in") || strings.Contains(err.Error(), "holds the line") {
		t.Fatalf("err = %v", err)
	}
	commands := strings.Join(shell.commands, "\n")
	checkoutBack := in(home, "bravo", "git checkout "+quote(runtime.GOOS, "main"))
	deleted := in(home, "bravo", "git branch -D tracker-line")
	if !strings.Contains(commands, checkoutBack+"\n"+deleted) || strings.Contains(commands, "push -u origin") {
		t.Fatalf("commands:\n%s", commands)
	}
	if got := read(t, filepath.Join(home, "code", "bravo", "AGENTS.md")); got != "# Bravo\n\nTracker: github-issues\n\nSome text.\n" {
		t.Fatalf("bravo AGENTS.md = %q", got)
	}
}

func TestOriginGhDoesNotKnowGetsNoBranchAndNoPush(t *testing.T) {
	home := homeWithRepos(t)
	savedRepos(t, home)
	shell := withRoast("1.1.0")
	shell.failing = map[string]bool{in(home, "bravo", "gh repo view --json name "+quote(runtime.GOOS, "git@github.com:me/bravo.git")): true}
	opts, out := options(t, home, shell, "")
	err := guided(t, opts, script{trackers: answering("Tracker: github-issues", "bravo")})
	if err == nil || !strings.Contains(err.Error(), "gh doesn't know the origin, so no PR was opened") {
		t.Fatalf("err = %v", err)
	}
	commands := strings.Join(shell.commands, "\n")
	if strings.Contains(commands, "checkout -b") || strings.Contains(commands, "push -u origin") {
		t.Fatalf("commands:\n%s", commands)
	}
	expectAll(t, out.String(), "bravo  FAILED: `gh repo view --json name ")
}
