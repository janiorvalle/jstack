package setup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// trackerRepo is one git checkout under a repos folder, as found on this
// run. file is the instructions file that carries the Tracker line, or the
// one it would be written to: AGENTS.md, or CLAUDE.md when that is the only
// one the repo keeps, with a link followed one hop, since AGENTS.md is
// often a link to CLAUDE.md and the write has to land in the file git
// tracks. line is the whole Tracker line, or "" when the repo declares
// none. hold is why an undeclared repo isn't asked about, "" when it is:
// the line waits on the tracker-line branch an earlier run made, so
// asking again would make a second one, the file links to outside the
// repo, where a line would speak for every repo that shares it, or the
// person skipped it on an earlier run, which skipped records.
type trackerRepo struct {
	dir     string
	name    string
	file    string
	line    string
	hold    string
	skipped bool
}

func (r trackerRepo) declared() bool {
	return r.line != ""
}

// askable is a repo the tracker question is for this run.
func (r trackerRepo) askable() bool {
	return !r.declared() && r.hold == ""
}

// insideRepo is the path git add takes for the file, relative to the
// checkout, and false when the file is a link to somewhere outside it.
// Both sides are resolved all the way, the way the write resolves the
// file, so a chain of links ends where the write would land.
func insideRepo(dir, file string) (string, bool) {
	relative, err := filepath.Rel(canonical(dir), canonical(file))
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(relative), true
}

// canonical is the path with every link followed. A file that isn't there
// yet resolves through its folder, so it compares with a folder that is.
func canonical(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	parent := filepath.Dir(path)
	if parent == path {
		return path
	}
	return filepath.Join(canonical(parent), filepath.Base(path))
}

// trackerBranch is the branch the PR offer commits the line on.
const trackerBranch = "tracker-line"

// reposReport is the repos half of the plan: the folders the person named
// and every checkout one level down each of them. guesses are the folders
// under home that look like repos folders, for the hint when none is named.
type reposReport struct {
	dirs    []string
	repos   []trackerRepo
	guesses []string
}

// undeclared is every repo the question is for this run.
func (r reposReport) undeclared() []trackerRepo {
	var repos []trackerRepo
	for _, repo := range r.repos {
		if repo.askable() {
			repos = append(repos, repo)
		}
	}
	return repos
}

// backend is one of the trackers the tracker skill knows. argument names
// the one thing the backend needs after its key, "" when it needs nothing;
// example is what the question shows, and byDefault is what Enter picks.
type backend struct {
	key       string
	label     string
	argument  string
	example   string
	byDefault string
}

// gitHubIssues is the one backend an origin can suggest.
const gitHubIssues = "github-issues"

// backends in the order the screens come: the one that needs a key first,
// then the one the origin can guess, then the folder in the repo for the
// rest.
var backends = []backend{
	{key: "linear", label: "Linear", argument: "team key", example: "SR"},
	{key: gitHubIssues, label: "GitHub Issues"},
	{key: "markdown", label: "markdown tasks in the repo", argument: "folder", example: "tasks/", byDefault: "tasks/"},
}

// trackerLine is the line the tracker skill reads, "Tracker: linear SR".
func trackerLine(chosen backend, argument string) string {
	if argument == "" {
		return "Tracker: " + chosen.key
	}
	return "Tracker: " + chosen.key + " " + argument
}

// reposFolderNames are the folders under home people keep checkouts in.
var reposFolderNames = []string{"code", "github", "src", "projects", "dev"}

// guessReposDirs is every one of those folders that holds a checkout.
func guessReposDirs(home string) []string {
	var found []string
	for _, name := range reposFolderNames {
		dir := filepath.Join(home, name)
		if len(scanDir(dir, nil)) > 0 {
			found = append(found, dir)
		}
	}
	return found
}

// chooseReposDirs is the folders this run scans: the saved ones, then the
// ones --repos-dir names, resolved and checked the way a typed answer is,
// so the config never holds a path that means something else from another
// working directory. A flag also settles the question, the way a
// --skill-repo does.
func chooseReposDirs(config Config, opts Options) (dirs []string, asked bool, err error) {
	flagged, err := parseReposDirs(strings.Join(opts.ReposDirs, ","), opts.Home)
	if err != nil {
		return nil, false, err
	}
	seen := map[string]bool{}
	for _, dir := range append(append([]string{}, config.ReposDirs...), flagged...) {
		if !seen[dir] {
			seen[dir] = true
			dirs = append(dirs, dir)
		}
	}
	return dirs, config.ReposDirsAsked || len(opts.ReposDirs) > 0, nil
}

// parseReposDirs turns what the person typed into folders that exist. A
// leading ~ is home, and a relative path is taken from where setup runs.
func parseReposDirs(answer, home string) ([]string, error) {
	var dirs []string
	for _, field := range strings.Split(answer, ",") {
		path := strings.TrimSpace(field)
		if path == "" {
			continue
		}
		if path == "~" || strings.HasPrefix(path, "~/") {
			path = filepath.Join(home, strings.TrimPrefix(path, "~"))
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("[SQUIRREL-REPOS-DIR] cannot resolve %q: %v; type a full path such as %s", path, err, filepath.Join(home, "code"))
		}
		if !isDir(absolute) {
			return nil, fmt.Errorf("[SQUIRREL-REPOS-DIR] %q is not a folder; type the path of a folder that exists, such as %s", path, filepath.Join(home, "code"))
		}
		dirs = append(dirs, absolute)
	}
	return dirs, nil
}

// scanRepos reads every named folder. The disk is the only source: no
// network, no git command, so the plan costs one directory listing per
// folder and two small reads per checkout. skipped is the checkouts an
// earlier run skipped, by canonical path.
func scanRepos(dirs []string, skipped map[string]bool) []trackerRepo {
	var repos []trackerRepo
	for _, dir := range dirs {
		repos = append(repos, scanDir(dir, skipped)...)
	}
	return repos
}

// scanDir lists the checkouts one level down: every folder with a .git in
// it, a folder for a clone or a file for a worktree.
func scanDir(dir string, skipped map[string]bool) []trackerRepo {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var repos []trackerRepo
	for _, entry := range entries {
		checkout := filepath.Join(dir, entry.Name())
		if !entry.IsDir() || !exists(filepath.Join(checkout, ".git")) {
			continue
		}
		file, line := readTracker(checkout)
		skippedEarlier := skipped[canonical(checkout)]
		repos = append(repos, trackerRepo{dir: checkout, name: entry.Name(), file: file, line: line, hold: holdReason(checkout, file, line, skippedEarlier), skipped: skippedEarlier})
	}
	sort.Slice(repos, func(left, right int) bool { return repos[left].name < repos[right].name })
	return repos
}

// holdReason is why an undeclared repo isn't asked about, in the words the
// report shows, or "" when it is.
func holdReason(checkout, file, line string, skipped bool) string {
	if line != "" {
		return ""
	}
	if skipped {
		return "skipped on an earlier run; add --ask-trackers-again to be asked"
	}
	if hasBranch(checkout, trackerBranch) {
		return "the line waits on branch " + trackerBranch + " until its PR merges; delete the branch to be asked again"
	}
	if _, inside := insideRepo(checkout, file); !inside {
		return filepath.Base(checkout) + "'s instructions file links to outside the repo, so setup leaves it alone"
	}
	return ""
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// hasBranch reads the branch off the disk, so a repo whose PR is still open
// is seen without running git: the loose ref git writes for a new branch,
// or the packed-refs file once gc has packed it, in the common git dir,
// which a linked worktree names through its .git file and commondir.
func hasBranch(checkout, name string) bool {
	common := gitCommonDir(checkout)
	if exists(filepath.Join(common, "refs", "heads", name)) {
		return true
	}
	packed, err := os.ReadFile(filepath.Join(common, "packed-refs"))
	return err == nil && strings.Contains(string(packed), " refs/heads/"+name+"\n")
}

// gitCommonDir is where a checkout's branches live: .git itself for a
// clone, and for a linked worktree the folder its .git file names, then
// the one that folder's commondir file names.
func gitCommonDir(checkout string) string {
	gitPath := filepath.Join(checkout, ".git")
	content, err := os.ReadFile(gitPath)
	if err != nil {
		return gitPath
	}
	gitDir := resolveGitPath(checkout, strings.TrimPrefix(strings.TrimSpace(string(content)), "gitdir:"))
	common, err := os.ReadFile(filepath.Join(gitDir, "commondir"))
	if err != nil {
		return gitDir
	}
	return resolveGitPath(gitDir, string(common))
}

func resolveGitPath(base, path string) string {
	path = strings.TrimSpace(path)
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(base, path)
}

// readTracker finds the Tracker line in AGENTS.md, then CLAUDE.md, and says
// which file a missing line would go into: AGENTS.md when the repo has one
// or has neither, CLAUDE.md when that is the only one it keeps. A file
// that is a link is named by what it points to.
func readTracker(dir string) (file, line string) {
	target := ""
	for _, name := range []string{"AGENTS.md", "CLAUDE.md"} {
		path := followLink(filepath.Join(dir, name))
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if line := findTrackerLine(string(content)); line != "" {
			return path, line
		}
		if target == "" {
			target = path
		}
	}
	if target == "" {
		target = filepath.Join(dir, "AGENTS.md")
	}
	return target, ""
}

// followLink is the file a link points to, one hop, relative to the link's
// folder, and the path itself when it isn't a link.
func followLink(path string) string {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return path
	}
	target, err := os.Readlink(path)
	if err != nil {
		return path
	}
	if filepath.IsAbs(target) {
		return filepath.Clean(target)
	}
	return filepath.Join(filepath.Dir(path), target)
}

func findTrackerLine(content string) string {
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "Tracker:") {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

// withTrackerLine puts the line on its own line after the heading the
// file opens with, a blank line on each side, or first when the file opens
// with anything else.
func withTrackerLine(content, line string) string {
	if content == "" {
		return line + "\n"
	}
	if !strings.HasPrefix(content, "#") {
		return line + "\n\n" + content
	}
	heading, rest, _ := strings.Cut(content, "\n")
	insert := "\n" + line + "\n"
	if rest != "" && !strings.HasPrefix(rest, "\n") {
		insert += "\n"
	}
	return heading + "\n" + insert + rest
}

func printReposPlan(out io.Writer, home string, report reposReport) {
	if len(report.dirs) == 0 {
		fmt.Fprintln(out, "\nrepos")
		hint := "  none named, so the trackers are not checked; add --repos-dir <folder> to name where your repos live"
		if len(report.guesses) > 0 {
			hint += ", found " + displayList(home, report.guesses)
		}
		fmt.Fprintln(out, hint)
		return
	}
	width := 0
	for _, repo := range report.repos {
		width = max(width, len(repo.name))
	}
	for _, dir := range report.dirs {
		fmt.Fprintf(out, "\nrepos %s\n", display(home, dir))
		count := 0
		for _, repo := range report.repos {
			if filepath.Dir(repo.dir) != dir {
				continue
			}
			count++
			state := "not declared"
			switch {
			case repo.declared():
				state = repo.line
			case repo.hold != "":
				state = "not declared, " + repo.hold
			}
			fmt.Fprintf(out, "  %-*s  %s\n", width, repo.name, state)
		}
		if count == 0 {
			fmt.Fprintln(out, "  no git checkout one level down")
		}
	}
}

// printTrackerAnswers is the repos half of the plan's answers: the line
// each undeclared repo gets and whether a PR can carry it, with the reason
// when it can't, or skipped, which the next run remembers. The PR question
// comes after the plan, so a line a PR can carry says so and the answer
// decides.
func printTrackerAnswers(out io.Writer, answers []TrackerAnswer, questions []TrackerQuestion) {
	if len(answers) == 0 {
		return
	}
	holds := map[string]string{}
	for _, question := range questions {
		holds[question.Dir] = question.PRHold
	}
	width := 0
	for _, answer := range answers {
		width = max(width, len(answer.Repo))
	}
	fmt.Fprintln(out, "\ntrackers")
	for _, answer := range answers {
		state := "skipped; not offered again without --ask-trackers-again"
		switch hold := holds[answer.Dir]; {
		case answer.Skip:
		case hold == "":
			state = fmt.Sprintf("write %q, open a PR on branch %s", answer.Line, trackerBranch)
		default:
			state = fmt.Sprintf("write %q only, %s", answer.Line, hold)
		}
		fmt.Fprintf(out, "  %-*s  %s\n", width, answer.Repo, state)
	}
}

// rememberSkips is the checkouts the next run leaves out of the tracker
// screens, by canonical path: the ones skipped on an earlier run and still
// held, and the ones skipped on this one. A repo that names its tracker
// since drops off, and so does one whose folder is gone.
func rememberSkips(report reposReport, answers []TrackerAnswer) []string {
	skippedNow := map[string]bool{}
	for _, answer := range answers {
		skippedNow[answer.Dir] = answer.Skip
	}
	var skipped []string
	for _, repo := range report.repos {
		if !repo.declared() && (repo.skipped || skippedNow[repo.dir]) {
			skipped = append(skipped, canonical(repo.dir))
		}
	}
	return skipped
}

// applyTrackers writes each answer into its repo. The harnesses are done by
// now, so a repo that fails is reported at the end instead of stopping the
// run. Without answers, the run had no terminal: the undeclared repos are
// listed and nothing is written, since someone's repo is their code.
func applyTrackers(ctx context.Context, opts Options, report reposReport, answers []TrackerAnswer) error {
	out := opts.Stdout
	undeclared := report.undeclared()
	if len(report.dirs) == 0 {
		return nil
	}
	fmt.Fprintln(out, "\nrepos")
	if len(undeclared) == 0 {
		fmt.Fprintln(out, "  every repo names its tracker")
		return nil
	}
	if len(answers) == 0 {
		fmt.Fprintf(out, "  %d not declared: %s; rerun with a terminal to name the tracker of each one\n", len(undeclared), repoNames(undeclared))
		return nil
	}
	byDir := map[string]trackerRepo{}
	for _, repo := range report.repos {
		byDir[repo.dir] = repo
	}
	var failures []error
	for _, answer := range answers {
		repo, ok := byDir[answer.Dir]
		if !ok {
			continue
		}
		if repo.declared() {
			fmt.Fprintf(out, "  %s  names its tracker since the question, %q, so the answer is left out\n", repo.name, repo.line)
			continue
		}
		if answer.Skip {
			fmt.Fprintf(out, "  %s  skipped\n", repo.name)
			continue
		}
		if err := declareTracker(ctx, opts, repo, answer.Line, answer.OpenPR); err != nil {
			fmt.Fprintf(out, "  %s  FAILED: %v\n", repo.name, err)
			failures = append(failures, fmt.Errorf("%s: %w", repo.name, err))
		}
	}
	if len(failures) == 0 {
		return nil
	}
	return fmt.Errorf("[SQUIRREL-TRACKERS] the harnesses are done, but %d repo step(s) failed:\n%w", len(failures), errors.Join(failures...))
}

// declareTracker writes the line into the repo, then opens the PR the
// person asked for when the tree had nothing else pending, the repo has an
// origin to push to, its push URL since that is where the branch goes, and
// it sits on its default branch, so the PR carries the line and nothing
// else. Every check runs before the write, so the line itself never counts
// as the pending change, and again here, since the answer was given a few
// screens ago.
func declareTracker(ctx context.Context, opts Options, repo trackerRepo, line string, openPR bool) error {
	out := opts.Stdout
	state := readRepoState(ctx, opts, repo.dir)
	content, err := os.ReadFile(repo.file)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("cannot read %q: %w; make it readable and rerun", repo.file, err)
	}
	if declared := findTrackerLine(string(content)); declared != "" {
		fmt.Fprintf(out, "  %s  names its tracker since the question, %q, so the answer is left out\n", repo.name, declared)
		return nil
	}
	if err := writeFile(repo.file, withTrackerLine(string(content), line)); err != nil {
		return err
	}
	fmt.Fprintf(out, "  %s  wrote %q to %s\n", repo.name, line, display(opts.Home, repo.file))
	if hold := state.prHold(); hold != "" {
		fmt.Fprintf(out, "  %s  %s; the line is left uncommitted, commit it yourself\n", repo.name, hold)
		return nil
	}
	if !openPR {
		fmt.Fprintf(out, "  %s  line left uncommitted; commit it yourself\n", repo.name)
		return nil
	}
	return openTrackerPR(ctx, opts, repo, state)
}

// repoState is what decides the PR offer, read through git before the
// write. defaultBranch is what origin/HEAD names, "" when the clone never
// recorded one, and a guess would decide which commits the PR carries, so
// there is no guess. published is HEAD at the remote's copy of that
// branch, so the PR carries no commit that was only ever local.
type repoState struct {
	clean         bool
	origin        string
	branch        string
	defaultBranch string
	published     bool
}

func (s repoState) hasRemote() bool {
	return s.origin != ""
}

func (s repoState) onDefaultBranch() bool {
	return s.defaultBranch != "" && s.branch == s.defaultBranch
}

// prHold is why git says the PR can't carry the line, in the words the
// plan and the report show, "" when every check passes.
func (s repoState) prHold() string {
	switch {
	case !s.clean:
		return "has other uncommitted changes"
	case !s.hasRemote():
		return "has no origin remote"
	case s.defaultBranch == "":
		return "has no origin/HEAD, run `git remote set-head origin -a` there to name its default branch"
	case !s.onDefaultBranch():
		return fmt.Sprintf("is on branch %s, not %s", s.branch, s.defaultBranch)
	case !s.published:
		return fmt.Sprintf("is on %s but not at origin/%s, push or pull first", s.branch, s.branch)
	}
	return ""
}

// prHold is why the PR can't carry the line: what git says, then whether
// gh can see the origin, since gh opens the PR. What gh said is remembered
// between runs, so the fix names the flag that asks again.
func prHold(state repoState, origin OriginFacts) string {
	if hold := state.prHold(); hold != "" {
		return hold
	}
	if !origin.Seen {
		return "has an origin gh can't see, run gh auth login for that host and add --ask-trackers-again"
	}
	return ""
}

// OriginFacts is what gh said about an origin, and when: Seen when gh
// could reach the repo, so it is on a GitHub host gh is logged into, and
// Issues when the repo has issues enabled, the one thing an origin says
// about which tracker the repo uses. The config keeps them by push URL, so
// a rerun asks gh only about origins it hasn't met.
type OriginFacts struct {
	Seen    bool      `json:"seen"`
	Issues  bool      `json:"issues"`
	AskedAt time.Time `json:"asked_at"`
}

// guess is the backend key the origin suggests, "" when it says nothing.
func (o OriginFacts) guess() string {
	if o.Seen && o.Issues {
		return gitHubIssues
	}
	return ""
}

// readOrigin asks gh about the origin. Any failure, gh not logged in, a
// host gh doesn't know, no network, reads as an origin gh can't see.
func readOrigin(ctx context.Context, opts Options, url string) OriginFacts {
	facts := OriginFacts{AskedAt: opts.Now().UTC().Truncate(time.Second)}
	var out bytes.Buffer
	if opts.Shell(ctx, "gh repo view --json hasIssuesEnabled "+quote(runtime.GOOS, url), &out) != nil {
		return facts
	}
	var answer struct {
		HasIssuesEnabled bool `json:"hasIssuesEnabled"`
	}
	if err := json.Unmarshal(out.Bytes(), &answer); err != nil {
		return facts
	}
	facts.Seen, facts.Issues = true, answer.HasIssuesEnabled
	return facts
}

func readRepoState(ctx context.Context, opts Options, dir string) repoState {
	var status, origin, branch, head bytes.Buffer
	state := repoState{}
	state.clean = opts.Shell(ctx, inRepo(runtime.GOOS, dir, "git status --porcelain"), &status) == nil && strings.TrimSpace(status.String()) == ""
	if opts.Shell(ctx, inRepo(runtime.GOOS, dir, "git remote get-url --push origin"), &origin) == nil {
		state.origin = withoutCredentials(strings.TrimSpace(origin.String()))
	}
	if opts.Shell(ctx, inRepo(runtime.GOOS, dir, "git rev-parse --abbrev-ref HEAD"), &branch) == nil {
		state.branch = strings.TrimSpace(branch.String())
	}
	if opts.Shell(ctx, inRepo(runtime.GOOS, dir, "git symbolic-ref --short refs/remotes/origin/HEAD"), &head) == nil {
		state.defaultBranch = strings.TrimPrefix(strings.TrimSpace(head.String()), "origin/")
	}
	if !state.onDefaultBranch() {
		return state
	}
	var commits bytes.Buffer
	if opts.Shell(ctx, inRepo(runtime.GOOS, dir, "git rev-parse HEAD "+quote(runtime.GOOS, "refs/remotes/origin/"+state.branch)), &commits) == nil {
		lines := strings.Fields(commits.String())
		state.published = len(lines) == 2 && lines[0] == lines[1]
	}
	return state
}

// withoutCredentials is the URL with any password before the host dropped.
// A push URL can carry a token, https://me:ghp_x@github.com/me/app, and
// the URL names the origin in the config and in the gh calls, where only
// the repo matters and a token must never land. The user name stays, since
// ssh://git@github.com/me/app is a different login without it, and an
// scp-style URL, git@github.com:me/app.git, has no scheme, so it parses as
// nothing and stays as it is.
func withoutCredentials(pushURL string) string {
	parsed, err := url.Parse(pushURL)
	if err != nil || parsed.User == nil {
		return pushURL
	}
	if _, hasPassword := parsed.User.Password(); !hasPassword {
		return pushURL
	}
	parsed.User = url.User(parsed.User.Username())
	return parsed.String()
}

// trackerPRBody is the ticket shape the tracker skill asks for.
const trackerPRBody = `Problem: This repo does not say where its work is tracked, so an agent starting a task here has nothing to claim against.
Fix: One Tracker line in the instructions file, the line the tracker skill reads.
Done when:
- grep -h ^Tracker: AGENTS.md CLAUDE.md prints the line.

Opened by squirrel setup.`

// openTrackerPR checks that gh knows the origin, by its URL so it is the
// remote the push goes to and not one gh prefers, commits the line on its
// own branch, pushes, opens the PR, and puts the repo back on the branch
// it was on, whether or not a step failed after the branch was made. A
// failure before the commit deletes the empty branch, so the next run
// asks again instead of reporting a PR that was never made. The push
// borrows gh as git's credential helper for that one command, and the PR
// goes through gh, so gh's login is what reaches GitHub either way and
// nothing in the person's git config changes.
func openTrackerPR(ctx context.Context, opts Options, repo trackerRepo, state repoState) error {
	out := opts.Stdout
	operatingSystem := runtime.GOOS
	previous := state.branch
	staged, _ := insideRepo(repo.dir, repo.file)
	steps := []string{
		"gh repo view --json name " + quote(operatingSystem, state.origin),
		"git checkout -b " + trackerBranch,
		"git add " + quote(operatingSystem, staged),
		"git commit -m " + quote(operatingSystem, "docs: name the tracker"),
		"git -c credential.helper=" + quote(operatingSystem, "!gh auth git-credential") + " push -u origin " + trackerBranch,
		"gh pr create --title " + quote(operatingSystem, "docs: name the tracker") + " --body " + quote(operatingSystem, trackerPRBody),
	}
	const branched, committed = 1, 3
	failedAt := -1
	var failed error
	for index, step := range steps {
		fmt.Fprintf(out, "  %s  %s\n", repo.name, step)
		if err := opts.Shell(ctx, inRepo(operatingSystem, repo.dir, step), out); err != nil {
			failedAt, failed = index, err
			break
		}
	}
	if failedAt == 0 {
		return fmt.Errorf("`%s` failed: %v; gh doesn't know the origin, so no PR was opened; the line is written and uncommitted, commit it yourself", steps[0], failed)
	}
	if failedAt == branched {
		return fmt.Errorf("`%s` failed: %v; the line is written and uncommitted in %s, commit it yourself", steps[branched], failed, repo.dir)
	}
	if err := opts.Shell(ctx, inRepo(operatingSystem, repo.dir, "git checkout "+quote(operatingSystem, previous)), out); err != nil {
		if failed == nil {
			return fmt.Errorf("the PR is open, but `git checkout %s` failed: %v; the repo is still on %s", previous, err, trackerBranch)
		}
		return fmt.Errorf("`%s` failed: %v, and `git checkout %s` failed after it: %v; the repo is on %s, sort it out by hand in %s", steps[failedAt], failed, previous, err, trackerBranch, repo.dir)
	}
	switch {
	case failed == nil:
		fmt.Fprintf(out, "  %s  PR opened, back on %s\n", repo.name, previous)
		return nil
	case failedAt <= committed:
		_ = opts.Shell(ctx, inRepo(operatingSystem, repo.dir, "git branch -D "+trackerBranch), out)
		return fmt.Errorf("`%s` failed: %v; the line is written and uncommitted in %s, commit it yourself", steps[failedAt], failed, repo.dir)
	}
	return fmt.Errorf("`%s` failed: %v; finish the PR by hand from %s, the branch %s holds the line", steps[failedAt], failed, repo.dir, trackerBranch)
}

// inRepo runs a command inside a checkout, and stops there when the folder
// can't be entered, so a command never runs in whatever folder setup was
// started from. The shape is the one the skills repo sync uses.
func inRepo(operatingSystem, dir, command string) string {
	if operatingSystem == "windows" {
		return "Set-Location " + quote(operatingSystem, dir) + " -ErrorAction Stop; " + command
	}
	return "cd " + quote(operatingSystem, dir) + " && " + command
}

func repoNames(repos []trackerRepo) string {
	names := make([]string, 0, len(repos))
	for _, repo := range repos {
		names = append(names, repo.name)
	}
	return strings.Join(names, ", ")
}

func displayList(home string, paths []string) string {
	shown := make([]string, 0, len(paths))
	for _, path := range paths {
		shown = append(shown, display(home, path))
	}
	return strings.Join(shown, ", ")
}
