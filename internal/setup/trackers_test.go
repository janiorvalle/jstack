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
	write(t, filepath.Join(home, ".squirrel", "config.json"), `{"harnesses":["claude"],"skill_repos_asked":true,"repos_dirs":["`+strings.ReplaceAll(filepath.Join(home, "code"), `\`, `\\`)+`"],"repos_dirs_asked":true}`)
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
			answers = append(answers, TrackerAnswer{Dir: question.Dir, Repo: question.Repo, Line: line, OpenPR: openPR && question.PRHold == ""})
		}
		return answers
	}
}

// answerEach is one answer per repo by name, with the Dir the question
// carries.
func answerEach(questions []TrackerQuestion, byName map[string]TrackerAnswer) []TrackerAnswer {
	var answers []TrackerAnswer
	for _, question := range questions {
		answer := byName[question.Repo]
		answer.Dir, answer.Repo = question.Dir, question.Repo
		answers = append(answers, answer)
	}
	return answers
}

// offers is which repos the questions offer the PR for, as "bravo:yes charlie:no".
func offers(questions []TrackerQuestion) string {
	var parts []string
	for _, question := range questions {
		state := "no"
		if question.PRHold == "" {
			state = "yes"
		}
		parts = append(parts, question.Repo+":"+state)
	}
	return strings.Join(parts, " ")
}

// backendNamed is the backend with that key.
func backendNamed(t *testing.T, key string) Backend {
	t.Helper()
	for _, entry := range Backends() {
		if entry.Key == key {
			return entry
		}
	}
	t.Fatalf("no backend %q", key)
	return Backend{}
}

// jsonPath is a path the way config.json holds it.
func jsonPath(path string) string {
	return strings.ReplaceAll(path, `\`, `\\`)
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
	repos := scanRepos([]string{filepath.Join(home, "code")}, nil)
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
	expectAll(t, out.String(), `echo  wrote "Tracker: github-issues" to ~/code/echo/CLAUDE.md`, "echo  has no origin remote; the line is left uncommitted")
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
		"\ntrackers\n  bravo    skipped; not offered again without --ask-trackers-again\n  charlie  skipped; not offered again without --ask-trackers-again\n",
		"bravo  skipped\n  charlie  skipped",
	)
	if got := read(t, filepath.Join(home, ".squirrel", "config.json")); !strings.Contains(got, `"repos_dirs": [`) || !strings.Contains(got, `"repos_dirs_asked": true`) {
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
	expectAll(t, out.String(), "repos ~/code\n  alpha    Tracker: markdown tasks/\n  bravo    not declared, skipped on an earlier run; add --ask-trackers-again to be asked\n")
}

func TestTypedReposFolderMustExist(t *testing.T) {
	home := homeWithRepos(t)
	if err := os.Rename(filepath.Join(home, "code"), filepath.Join(home, "work")); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseReposDirs("nowhere", home); err == nil || !strings.Contains(err.Error(), `[SQUIRREL-REPOS-DIR] "nowhere" is not a folder; type the path of a folder that exists, such as `+filepath.Join(home, "code")) {
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
	if got := read(t, filepath.Join(home, ".squirrel", "config.json")); !strings.Contains(got, `"repos_dirs_asked": true`) {
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
	expectAll(t, out.String(), "squirrel setup --harness claude --yes --repos-dir "+quote(runtime.GOOS, filepath.Join(home, "code")))
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
		linear, markdown := backendNamed(t, "linear"), backendNamed(t, "markdown")
		return answerEach(questions, map[string]TrackerAnswer{
			"bravo":   {Line: TrackerLine(linear, "SR"), OpenPR: true},
			"charlie": {Line: TrackerLine(markdown, markdown.Default)},
		})
	}})
	if err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(),
		"\ntrackers\n  bravo    write \"Tracker: linear SR\", open a PR on branch tracker-line\n  charlie  write \"Tracker: markdown tasks/\" only, has no origin remote\n",
		`bravo  wrote "Tracker: linear SR" to ~/code/bravo/AGENTS.md`,
		"bravo  PR opened, back on main",
		`charlie  wrote "Tracker: markdown tasks/" to ~/code/charlie/AGENTS.md`,
		"charlie  has no origin remote; the line is left uncommitted, commit it yourself",
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
	err := guided(t, opts, script{trackers: func(questions []TrackerQuestion) []TrackerAnswer {
		return answerEach(questions, map[string]TrackerAnswer{"bravo": {Skip: true}, "charlie": {Line: TrackerLine(backendNamed(t, "jira"), "SR")}})
	}})
	if err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "bravo    skipped; not offered again without --ask-trackers-again", "bravo  skipped", `charlie  wrote "Tracker: jira SR"`)
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
	expectAll(t, out.String(), `bravo  wrote "Tracker: github-issues"`, "bravo  has other uncommitted changes; the line is left uncommitted, commit it yourself", "charlie  line left uncommitted; commit it yourself")
}

func TestFailedPushGoesBackToTheBranchAndIsReportedAtTheEnd(t *testing.T) {
	home := homeWithRepos(t)
	savedRepos(t, home)
	shell := withRoast("1.1.0")
	shell.versions[in(home, "bravo", "git rev-parse --abbrev-ref HEAD")] = "main"
	shell.failing = map[string]bool{in(home, "bravo", "git -c credential.helper="+quote(runtime.GOOS, "!gh auth git-credential")+" push -u origin tracker-line"): true}
	opts, out := options(t, home, shell, "")
	err := guided(t, opts, script{trackers: answering("Tracker: github-issues", "bravo")})
	if err == nil || !strings.Contains(err.Error(), "[SQUIRREL-TRACKERS] the harnesses are done, but 1 repo step(s) failed") || !strings.Contains(err.Error(), "push -u origin tracker-line` failed") {
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
		return answerEach(questions, map[string]TrackerAnswer{"charlie": {Skip: true}})
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
	if got := read(t, filepath.Join(home, ".squirrel", "config.json")); !strings.Contains(got, strings.ReplaceAll(filepath.Join(home, "code"), `\`, `\\`)) || strings.Contains(got, `"code"`) {
		t.Fatalf("config = %q", got)
	}
	opts, _ = options(t, home, withRoast("1.1.0"), "")
	opts.ReposDirs = []string{"nowhere"}
	err := Run(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), `[SQUIRREL-REPOS-DIR] "nowhere" is not a folder`) {
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
	err := guided(t, opts, script{trackers: func(questions []TrackerQuestion) []TrackerAnswer {
		return answerEach(questions, map[string]TrackerAnswer{"bravo": {Line: "Tracker: github-issues", OpenPR: true}, "charlie": {Skip: true}})
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
		return answerEach(questions, map[string]TrackerAnswer{"bravo": {Skip: true}})
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
	expectAll(t, out.String(), "bravo  is on branch feature, not main; the line is left uncommitted, commit it yourself", "charlie  has no origin/HEAD, run `git remote set-head origin -a` there to name its default branch; the line is left uncommitted, commit it yourself")
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
	expectAll(t, out.String(), "bravo  is on main but not at origin/main, push or pull first; the line is left uncommitted, commit it yourself", "charlie  line left uncommitted; commit it yourself")
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

func TestTwoReposFoldersWithTheSameRepoNameEachGetTheirOwnAnswer(t *testing.T) {
	home := homeWithRepos(t)
	work := filepath.Join(home, "work")
	if err := os.MkdirAll(filepath.Join(work, "bravo", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(home, ".squirrel", "config.json"), `{"harnesses":["claude"],"skill_repos_asked":true,"repos_dirs":["`+strings.ReplaceAll(filepath.Join(home, "code"), `\`, `\\`)+`","`+strings.ReplaceAll(work, `\`, `\\`)+`"],"repos_dirs_asked":true}`)
	opts, out := options(t, home, withRoast("1.1.0"), "")
	err := guided(t, opts, script{trackers: func(questions []TrackerQuestion) []TrackerAnswer {
		var answers []TrackerAnswer
		for _, question := range questions {
			answer := TrackerAnswer{Dir: question.Dir, Repo: question.Repo, Skip: true}
			if question.Dir == filepath.Join(work, "bravo") {
				answer = TrackerAnswer{Dir: question.Dir, Repo: question.Repo, Line: "Tracker: jira SR"}
			}
			answers = append(answers, answer)
		}
		return answers
	}})
	if err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), `bravo  wrote "Tracker: jira SR" to ~/work/bravo/AGENTS.md`)
	if read(t, filepath.Join(home, "code", "bravo", "AGENTS.md")) != "# Bravo\n\nSome text.\n" {
		t.Fatal("the answer for ~/work/bravo landed in ~/code/bravo")
	}
	if got := read(t, filepath.Join(work, "bravo", "AGENTS.md")); got != "Tracker: jira SR\n" {
		t.Fatalf("~/work/bravo AGENTS.md = %q", got)
	}
}

func TestATrackerAnswerAloneIsPending(t *testing.T) {
	home := homeWithRepos(t)
	savedRepos(t, home)
	opts, _ := options(t, home, withRoast("1.1.0"), "")
	if err := guided(t, opts, script{}); err != nil {
		t.Fatal(err)
	}
	opts.AskTrackersAgain = true
	session, err := Start(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if _, err := session.Gather(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	dirs, _ := session.ReposDirs()
	questions := session.Trackers(context.Background(), dirs)
	answers := Answers{Harnesses: []string{"claude"}, ReposDirs: dirs, Tools: map[string]bool{}}
	for _, question := range questions {
		answers.Trackers = append(answers.Trackers, TrackerAnswer{Dir: question.Dir, Repo: question.Repo, Skip: true})
	}
	plan, err := session.Plan(context.Background(), answers)
	if err != nil || plan.Pending() {
		t.Fatalf("every repo skipped is pending: err = %v", err)
	}
	answers.Trackers[0].Skip, answers.Trackers[0].Line = false, "Tracker: github-issues"
	if plan, err = session.Plan(context.Background(), answers); err != nil || !plan.Pending() {
		t.Fatalf("a line to write is not pending: err = %v", err)
	}
}

func TestATrackerLineWrittenSinceTheQuestionIsLeftAlone(t *testing.T) {
	home := homeWithRepos(t)
	savedRepos(t, home)
	opts, out := options(t, home, withRoast("1.1.0"), "")
	err := guided(t, opts, script{trackers: func(questions []TrackerQuestion) []TrackerAnswer {
		write(t, filepath.Join(home, "code", "bravo", "AGENTS.md"), "# Bravo\n\nTracker: linear SR\n\nSome text.\n")
		return answerEach(questions, map[string]TrackerAnswer{"bravo": {Line: "Tracker: github-issues", OpenPR: true}, "charlie": {Skip: true}})
	}})
	if err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), `bravo  names its tracker since the question, "Tracker: linear SR", so the answer is left out`)
	if got := read(t, filepath.Join(home, "code", "bravo", "AGENTS.md")); got != "# Bravo\n\nTracker: linear SR\n\nSome text.\n" {
		t.Fatalf("bravo AGENTS.md = %q", got)
	}
	out.Reset()
	repo := trackerRepo{dir: filepath.Join(home, "code", "bravo"), name: "bravo", file: filepath.Join(home, "code", "bravo", "AGENTS.md")}
	if err := declareTracker(context.Background(), opts, repo, "Tracker: github-issues", true); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), `bravo  names its tracker since the question, "Tracker: linear SR", so the answer is left out`)
	if got := read(t, repo.file); got != "# Bravo\n\nTracker: linear SR\n\nSome text.\n" {
		t.Fatalf("the write went ahead: %q", got)
	}
}

func TestSkippedRepoIsRememberedUntilAskedForAgain(t *testing.T) {
	home := homeWithRepos(t)
	savedRepos(t, home)
	bravo := jsonPath(canonical(filepath.Join(home, "code", "bravo")))
	opts, out := options(t, home, withRoast("1.1.0"), "")
	err := guided(t, opts, script{trackers: func(questions []TrackerQuestion) []TrackerAnswer {
		return answerEach(questions, map[string]TrackerAnswer{"bravo": {Skip: true}, "charlie": {Line: "Tracker: github-issues"}})
	}})
	if err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "bravo    skipped; not offered again without --ask-trackers-again", "bravo  skipped")
	if got := read(t, filepath.Join(home, ".squirrel", "config.json")); !strings.Contains(got, "\"trackers_skipped\": [\n    \""+bravo+"\"\n  ]") {
		t.Fatalf("config = %s", got)
	}

	opts, out = options(t, home, withRoast("1.1.0"), "")
	err = guided(t, opts, script{trackers: func(questions []TrackerQuestion) []TrackerAnswer {
		if len(questions) != 0 {
			t.Fatalf("asked about a skipped repo: %+v", questions)
		}
		return nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "  bravo    not declared, skipped on an earlier run; add --ask-trackers-again to be asked\n  charlie  Tracker: github-issues\n")
	if got := read(t, filepath.Join(home, ".squirrel", "config.json")); !strings.Contains(got, bravo) {
		t.Fatalf("a run that asked nothing forgot the skip: %s", got)
	}

	opts, out = options(t, home, withRoast("1.1.0"), "")
	opts.AskTrackersAgain = true
	err = guided(t, opts, script{trackers: func(questions []TrackerQuestion) []TrackerAnswer {
		if got := offers(questions); got != "bravo:yes" {
			t.Fatalf("offers = %q", got)
		}
		return answerEach(questions, map[string]TrackerAnswer{"bravo": {Line: "Tracker: linear SR"}})
	}})
	if err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), `bravo  wrote "Tracker: linear SR"`)
	if got := read(t, filepath.Join(home, ".squirrel", "config.json")); strings.Contains(got, "trackers_skipped") {
		t.Fatalf("a declared repo stayed on the skip list: %s", got)
	}
}

func TestSkippedRepoThatNamesItsTrackerDropsOffTheList(t *testing.T) {
	home := homeWithRepos(t)
	savedRepos(t, home)
	opts, _ := options(t, home, withRoast("1.1.0"), "")
	if err := guided(t, opts, script{}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(home, ".squirrel", "config.json")); !strings.Contains(got, "trackers_skipped") {
		t.Fatalf("config = %s", got)
	}
	write(t, filepath.Join(home, "code", "bravo", "AGENTS.md"), "# Bravo\n\nTracker: jira SR\n")
	opts, out := options(t, home, withRoast("1.1.0"), "")
	opts.Yes = true
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	expectAll(t, out.String(), "  bravo    Tracker: jira SR\n  charlie  not declared, skipped on an earlier run; add --ask-trackers-again to be asked\n")
	got := read(t, filepath.Join(home, ".squirrel", "config.json"))
	if strings.Contains(got, jsonPath(canonical(filepath.Join(home, "code", "bravo")))) || !strings.Contains(got, jsonPath(canonical(filepath.Join(home, "code", "charlie")))) {
		t.Fatalf("config = %s", got)
	}
}

func TestOriginIsAskedOnceAndSaysWhetherIssuesAreOn(t *testing.T) {
	home := homeWithRepos(t)
	savedRepos(t, home)
	shell := withRoast("1.1.0")
	shell.versions[in(home, "charlie", "git remote get-url --push origin")] = "git@github.com:me/bravo.git"
	view := "gh repo view --json hasIssuesEnabled " + quote(runtime.GOOS, "git@github.com:me/bravo.git")
	ask := func() (string, string, int) {
		t.Helper()
		opts, _ := options(t, home, shell, "")
		session, err := Start(opts)
		if err != nil {
			t.Fatal(err)
		}
		defer session.Close()
		dirs, _ := session.ReposDirs()
		questions := session.Trackers(context.Background(), dirs)
		questions = append(questions, session.Trackers(context.Background(), dirs)...)
		var guesses, holds []string
		for _, question := range questions {
			guesses = append(guesses, question.Guess)
			holds = append(holds, question.PRHold)
		}
		calls := 0
		for _, command := range shell.commands {
			if command == view {
				calls++
			}
		}
		shell.commands = nil
		return strings.Join(guesses, ","), strings.Join(holds, ","), calls
	}
	if guesses, holds, calls := ask(); guesses != "github-issues,github-issues,github-issues,github-issues" || holds != ",,," || calls != 1 {
		t.Fatalf("issues on: guesses = %q, holds = %q, gh asked %d times", guesses, holds, calls)
	}
	shell.versions[view] = `{"hasIssuesEnabled":false}`
	if guesses, holds, _ := ask(); guesses != ",,," || holds != ",,," {
		t.Fatalf("issues off: guesses = %q, holds = %q", guesses, holds)
	}
	shell.failing = map[string]bool{view: true}
	hold := "has an origin gh can't see, run gh auth login for that host"
	if guesses, holds, _ := ask(); guesses != ",,," || holds != strings.Repeat(hold+",", 3)+hold {
		t.Fatalf("gh failing: guesses = %q, holds = %q", guesses, holds)
	}
}
