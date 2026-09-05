package setup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/janiorvalle/jstack/internal/harness"
	"github.com/janiorvalle/jstack/internal/letter"
	"github.com/janiorvalle/jstack/internal/skills"
	"github.com/janiorvalle/jstack/internal/tools"
)

// Session is one run of setup: the embedded assets, the saved config, and
// the harness table, read once, plus the facts a guided flow asks for as
// its screens come up. Each fact is computed the first time it's asked for
// and kept, so a screen shown again after Esc costs nothing. Run and the
// guided flow both go through it, so every rule lives here once.
type Session struct {
	opts     Options
	embedded assets
	config   Config
	rows     harness.Table

	gathered    catalog
	gatheredFor string
	gatheredYet bool
	checked     []toolStatus
	checkedYet  bool
	scanned     []TrackerQuestion
	scannedFor  string
	scannedYet  bool
}

// Start reads what every screen needs. Nothing on the machine changes.
func Start(opts Options) (*Session, error) {
	embedded, err := loadAssets(opts.Files)
	if err != nil {
		return nil, err
	}
	config, err := loadConfig(opts.Home)
	if err != nil {
		return nil, err
	}
	return &Session{opts: opts, embedded: embedded, config: config, rows: harness.Resolve(opts.Home, opts.Getenv)}, nil
}

// HarnessChoice is one row of the harness screen. Checked is the pick to
// start from: --harness when given, else the saved picks plus every
// harness found, so one installed since the last run is offered.
type HarnessChoice struct {
	Key     string
	Name    string
	Where   string
	Found   bool
	Checked bool
}

// Harnesses is the harness screen's rows, in table order.
func (s *Session) Harnesses() ([]HarnessChoice, error) {
	picked, source, err := choose(s.opts, s.rows, s.config)
	if err != nil {
		return nil, err
	}
	var choices []HarnessChoice
	for _, entry := range s.rows {
		checked := contains(picked, entry) || source == fromConfig && entry.Installed()
		choices = append(choices, HarnessChoice{Key: entry.Key, Name: entry.Name, Where: rootWithVariable(s.opts.Home, entry), Found: entry.Installed(), Checked: checked})
	}
	return choices, nil
}

// SkillRepos is the skills repos this run installs from before any
// question: the saved ones and the ones the flags name.
func (s *Session) SkillRepos() ([]string, error) {
	return chooseRepos(s.config.SkillRepos, s.opts)
}

// SkillRepoAsked reports whether the skills repo question is settled: it
// was asked on an earlier run, --no-skill-repo said there is none, or a
// repo is already named.
func (s *Session) SkillRepoAsked() bool {
	names, err := s.SkillRepos()
	return s.config.SkillReposAsked || s.opts.NoSkillRepo || (err == nil && len(names) > 0)
}

// RepoName normalizes what a person types or pastes for a skills repo to
// owner/name, or says what shape was expected.
func RepoName(spec string) (string, error) {
	return repoName(spec)
}

// Gather fetches the repos, prints them, and settles every skill name more
// than one source holds with the --override flags and the saved picks. It
// returns the collisions those don't settle, for the guided flow to ask
// about, one screen each. Gathering happens once per set of names.
func (s *Session) Gather(ctx context.Context, repoNames []string) ([]skills.Collision, error) {
	key := strings.Join(repoNames, ",")
	if s.gatheredYet && s.gatheredFor == key {
		return s.gathered.open, nil
	}
	if s.gatheredYet {
		s.gathered.close()
	}
	s.gatheredYet, s.gatheredFor = true, key
	s.gathered = holdBack(buildCatalog(s.embedded, syncRepos(ctx, s.opts, repoNames)), s.config.SkillOverrides)
	printRepos(s.opts.Stdout, s.opts.Home, s.gathered.repos)
	printHeld(s.opts.Stdout, s.gathered, s.config.SkillOverrides)
	collisions, err := skills.Collisions(s.gathered.sources)
	if err != nil {
		return nil, err
	}
	s.gathered.collisions = collisions
	picks, open, err := settleCollisions(collisions, s.config.SkillOverrides, s.opts.Overrides)
	if err != nil {
		return nil, err
	}
	s.gathered.picks, s.gathered.open = picks, open
	return open, nil
}

// Rename is the pick that stops setup so the person can rename the folder
// in their repo instead of choosing.
const Rename = "rename"

// PickSources takes the answers for the open collisions, one source per
// skill name, and prints where every colliding name comes from this run.
func (s *Session) PickSources(picks map[string]string) error {
	for _, collision := range s.gathered.open {
		source := picks[collision.Name]
		if source == Rename {
			return renameStop(collision)
		}
		if !holds(collision, source) {
			return fmt.Errorf("[JSTACK-OVERRIDE] skill %q is not in %s; it is in %s", collision.Name, source, strings.Join(collision.Sources, " and "))
		}
		s.gathered.picks[collision.Name] = source
	}
	printOverrides(s.opts.Stdout, s.gathered.collisions, s.gathered.picks)
	return nil
}

// Display is a path the way the report shows it, ~ for home.
func Display(home, path string) string {
	return display(home, path)
}

// ReposDirs is the folders this run scans before any question: the saved
// ones and the ones --repos-dir names.
func (s *Session) ReposDirs() ([]string, error) {
	dirs, _, err := chooseReposDirs(s.config, s.opts)
	return dirs, err
}

// ReposDirsAsked reports whether the repos folder question is settled.
func (s *Session) ReposDirsAsked() bool {
	_, asked, err := chooseReposDirs(s.config, s.opts)
	return err == nil && asked
}

// ReposDirGuesses is every folder under home that looks like a repos
// folder, for the screen to offer.
func (s *Session) ReposDirGuesses() []string {
	return guessReposDirs(s.opts.Home)
}

// ParseReposDirs turns what a person typed into folders that exist, comma
// separated for more than one, ~ for home.
func ParseReposDirs(answer, home string) ([]string, error) {
	return parseReposDirs(answer, home)
}

// TrackerQuestion is one repo the tracker question is for this run. Dir
// is the checkout, what an answer names, since two repos folders can each
// hold a repo called app. Repo is its name, for the screen. File is where
// the line would be written, shown the way the report shows paths.
// PROffer is true when the repo could take the PR: a clean tree on its
// default branch at the remote's commit, with an origin gh can push to,
// read before any write so the line itself never counts as pending.
type TrackerQuestion struct {
	Dir     string
	Repo    string
	File    string
	PROffer bool
}

// Trackers scans the folders and reads each undeclared repo's state, once
// per set of folders.
func (s *Session) Trackers(ctx context.Context, reposDirs []string) []TrackerQuestion {
	key := strings.Join(reposDirs, ",")
	if s.scannedYet && s.scannedFor == key {
		return s.scanned
	}
	s.scannedYet, s.scannedFor = true, key
	s.scanned = nil
	for _, repo := range planRepos(s.opts.Home, reposDirs).undeclared() {
		state := readRepoState(ctx, s.opts, repo.dir)
		s.scanned = append(s.scanned, TrackerQuestion{Dir: repo.dir, Repo: repo.name, File: display(s.opts.Home, repo.file), PROffer: state.prPossible()})
	}
	return s.scanned
}

// Backend is one of the trackers the tracker skill knows. Argument names
// the one thing it needs after its key, "" when it needs nothing; Example
// is what the question shows and Default what Enter picks.
type Backend struct {
	Key      string
	Label    string
	Argument string
	Example  string
	Default  string
}

// Backends lists them in the order the question shows.
func Backends() []Backend {
	list := make([]Backend, 0, len(backends))
	for _, entry := range backends {
		list = append(list, Backend{Key: entry.key, Label: entry.label, Argument: entry.argument, Example: entry.example, Default: entry.byDefault})
	}
	return list
}

// TrackerLine is the line the tracker skill reads, "Tracker: linear SR".
func TrackerLine(chosen Backend, argument string) string {
	return trackerLine(backend{key: chosen.Key}, argument)
}

// TrackerAnswer is what to do about the undeclared repo at Dir: write
// Line, or skip it this run, and open the PR for the line when the repo
// can take one. Repo is its name, for the plan.
type TrackerAnswer struct {
	Dir    string
	Repo   string
	Line   string
	Skip   bool
	OpenPR bool
}

// ToolChoice is one tool on the tools screen. Actionable is true when
// setup has a line it could run: an install when missing, an update when
// outdated and the binary's owner has one. Label is that offer, and State
// the tool's line in the plan for the rest. Checked is what the flags
// already agreed to.
type ToolChoice struct {
	Title      string
	Label      string
	State      string
	Actionable bool
	Checked    bool
}

// Tools runs every check and version line and looks up the latest
// versions, once per run.
func (s *Session) Tools(ctx context.Context) []ToolChoice {
	statuses := s.toolStatuses(ctx)
	choices := make([]ToolChoice, 0, len(statuses))
	for _, status := range statuses {
		choice := ToolChoice{Title: status.tool.Title, State: toolState(status), Actionable: status.actionable(), Checked: agreedByFlag(s.opts, status)}
		if choice.Actionable {
			choice.Label = toolOffer(status)
		}
		choices = append(choices, choice)
	}
	return choices
}

func (s *Session) toolStatuses(ctx context.Context) []toolStatus {
	if s.checkedYet {
		return s.checked
	}
	s.checkedYet = true
	s.checked = checkTools(ctx, s.opts, s.embedded.tools)
	return s.checked
}

// Answers is everything a run decides, from the flags, the config, and the
// screens, in the values the plan and the apply take.
type Answers struct {
	Harnesses      []string
	SkillRepos     []string
	SkillRepoAsked bool
	ReposDirs      []string
	ReposDirsAsked bool
	Tools          map[string]bool
	Trackers       []TrackerAnswer
}

// Plan is what a run would do, ready to print and apply.
type Plan struct {
	picked   []harness.Harness
	current  plan
	trackers []TrackerAnswer
	embedded assets
	noted    bool
}

// Plan works out what the answers mean for every harness, tool, and repo.
// Nothing on the machine changes.
func (s *Session) Plan(ctx context.Context, answers Answers) (*Plan, error) {
	picked, err := s.rows.ByKeys(answers.Harnesses)
	if err != nil {
		return nil, err
	}
	harnessPlans, err := planHarnesses(s.opts, s.embedded, s.gathered, picked)
	if err != nil {
		return nil, err
	}
	current := plan{harnesses: harnessPlans, catalog: s.gathered, repos: planRepos(s.opts.Home, answers.ReposDirs), tools: append([]toolStatus(nil), s.toolStatuses(ctx)...)}
	for index := range current.tools {
		status := &current.tools[index]
		status.install = status.actionable() && answers.Tools[status.tool.Title]
	}
	markSkillPresence(s.opts, picked, current.tools)
	return &Plan{picked: picked, current: current, trackers: answers.Trackers, embedded: s.embedded}, nil
}

// Print shows the plan the way the report shows the outcome: each harness,
// the tools, the repos, and what the answers say about each undeclared
// one.
func (p *Plan) Print(out io.Writer, opts Options, answers Answers) {
	printPlan(out, opts.Home, p.embedded, p.current)
	p.noted = noteInstallFolderOffPath(opts, out)
	printReposPlan(out, opts.Home, p.current.repos)
	printTrackerAnswers(out, answers.Trackers)
}

// Pending reports whether applying the plan would change anything beyond
// setup's own files: a skill or letter to write, a tool to install or
// update, a tool skill to put in place, a tracker line to write.
func (p *Plan) Pending() bool {
	for _, entry := range p.current.harnesses {
		if entry.skills.Pending() || entry.letter.Outcome != letter.Same {
			return true
		}
	}
	for _, status := range p.current.tools {
		if status.install || status.present && status.tool.SkillInstall != "" && !status.skillPresent {
			return true
		}
	}
	for _, answer := range p.trackers {
		if !answer.Skip {
			return true
		}
	}
	return false
}

// Apply writes everything the plan says, in the order the report shows:
// the harnesses, then the config, then the tools, then the repos. The
// harnesses and the config come first so a tool or repo step that fails
// leaves the skills and the letter in place and is reported at the end.
func (s *Session) Apply(ctx context.Context, p *Plan, answers Answers) error {
	opts := s.opts
	out := opts.Stdout
	if len(p.picked) == 0 {
		fmt.Fprintln(out, "\nNo harness picked. Nothing changed in the harnesses. Pass --harness claude,codex to name one.")
		return nil
	}
	backupRoot, err := reserveBackup(opts.Home, opts.Now().Format("20060102-150405"), p.current)
	if err != nil {
		return err
	}
	if err := applyHarnesses(opts, s.embedded, p.current, backupRoot); err != nil {
		return err
	}
	saved := Config{
		Harnesses:       harness.Keys(p.picked),
		SkillRepos:      answers.SkillRepos,
		SkillReposAsked: answers.SkillRepoAsked,
		SkillOverrides:  rememberOverrides(s.config.SkillOverrides, p.current.catalog.unreachable(), p.current.catalog.picks),
		ReposDirs:       answers.ReposDirs,
		ReposDirsAsked:  answers.ReposDirsAsked,
	}
	if err := saveConfig(opts.Home, saved); err != nil {
		return err
	}
	if err := writeScripts(s.embedded, opts.Home); err != nil {
		return err
	}
	toolsErr := applyTools(ctx, opts, p.current, p.picked, backupRoot)
	trackersErr := applyTrackers(ctx, opts, p.current.repos, answers.Trackers)
	if !p.noted {
		noteInstallFolderOffPath(opts, out)
	}
	fmt.Fprintf(out, "\nharness picks saved to %s\nrestart the harness so the skills load.\n", display(opts.Home, configPath(opts.Home)))
	return errors.Join(toolsErr, trackersErr)
}

// Close lets go of the clone folders the session holds open.
func (s *Session) Close() {
	if s.gatheredYet {
		s.gathered.close()
	}
}

// checkTools runs each tool's check and version line, looks up the latest
// versions, and locates each outdated tool's binary to find who updates
// it.
func checkTools(ctx context.Context, opts Options, list []tools.Tool) []toolStatus {
	var statuses []toolStatus
	for _, tool := range list {
		status := toolStatus{tool: tool, present: opts.Shell(ctx, tool.Check, io.Discard) == nil}
		if status.present {
			status.installed = installedVersion(ctx, opts, tool)
		}
		statuses = append(statuses, status)
	}
	lookupLatest(ctx, opts, statuses)
	where := &locator{ctx: ctx, opts: opts}
	for index := range statuses {
		if statuses[index].outdated() {
			found := where.locate(statuses[index].tool)
			statuses[index].path, statuses[index].owner, statuses[index].formula = found.path, found.owner, found.formula
		}
	}
	return statuses
}
