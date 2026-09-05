// Package tui is setup on a terminal: one screen per question on charm's
// huh, saved answers preselected, a plan and a confirm at the end. It
// collects answers and renders; every rule, the plan, and the apply stay in
// the setup package.
package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/janiorvalle/jstack/internal/setup"
)

// ErrQuit is returned when the person leaves before the plan is applied:
// Ctrl-C anywhere, or Esc on the first screen.
var ErrQuit = errors.New("quit")

// Run asks each question in turn, prints the plan, and applies it after
// the confirm. Nothing in a harness or a repo of the person's changes
// before that; the skills repo clone under ~/.jstack/repos is synced as
// soon as it's named, since its collisions are the next question.
func Run(ctx context.Context, opts setup.Options) error {
	session, err := setup.Start(opts)
	if err != nil {
		return err
	}
	defer session.Close()
	return run(ctx, session, opts, terminal(opts.Stdin, opts.Stdout))
}

// direction is which way a step is entered: forward from the one before
// it, or backward from the one after, when Esc walks into a list of
// questions and the last one should show.
type direction int

const (
	forward direction = iota
	backward
)

// step is one question, or a list of them, or a computation between two
// questions. skipped means nothing was shown, so Esc walks over it.
type step func(direction) (outcome, error)

const skipped outcome = -1

// sequence runs steps front to back, or back to front when entered
// backward. back from a step runs the one before it; back off the front
// is back for the whole list.
func sequence(steps ...step) step {
	return func(entered direction) (outcome, error) {
		index, heading, shown := 0, forward, false
		if entered == backward {
			index, heading = len(steps)-1, backward
		}
		for index >= 0 && index < len(steps) {
			result, err := steps[index](heading)
			if err != nil {
				return result, err
			}
			switch result {
			case quit:
				return quit, nil
			case next:
				shown, heading = true, forward
			case back:
				heading = backward
			}
			if heading == forward {
				index++
			} else {
				index--
			}
		}
		if index < 0 {
			return back, nil
		}
		if !shown {
			return skipped, nil
		}
		return next, nil
	}
}

// flow is the run in progress: the session the facts come from, and the
// answers so far, kept across screens so Esc shows a question with the
// answer it had.
type flow struct {
	ctx     context.Context
	session *setup.Session
	opts    setup.Options
	out     io.Writer
	show    runner

	harnesses     []string
	harnessesInit bool
	skillRepo     string
	collisions    []collisionScratch
	reposPick     string
	reposTyped    string
	trackers      []trackerScratch
	tools         []string
	toolsInit     bool
	plan          *setup.Plan
	answers       setup.Answers
	confirmed     bool
}

type collisionScratch struct {
	name    string
	sources []string
	pick    string
}

// trackerScratch is one undeclared repo's answers as the screens collect
// them. backend is a backend key, or skip.
type trackerScratch struct {
	question setup.TrackerQuestion
	backend  string
	argument string
	openPR   bool
	forRest  bool
}

const (
	skip      = "skip"
	elsewhere = "elsewhere"
)

func run(ctx context.Context, session *setup.Session, opts setup.Options, show runner) error {
	f := &flow{ctx: ctx, session: session, opts: opts, out: opts.Stdout, show: show}
	result, err := sequence(
		f.harnessScreen,
		f.skillRepoScreen,
		f.collisionScreens,
		f.reposDirScreen,
		f.reposTypedScreen,
		f.trackerScreens,
		f.toolsScreen,
		f.planScreen,
	)(forward)
	if err != nil {
		return err
	}
	if result == quit || result == back {
		return ErrQuit
	}
	if !f.confirmed {
		fmt.Fprintln(f.out, "Nothing changed in the harnesses.")
		return nil
	}
	return f.session.Apply(f.ctx, f.plan, f.answers)
}

// ask runs one screen and reads how it ended.
func (f *flow) ask(field huh.Field, summary func() string) (outcome, error) {
	s := newScreen(field, summary)
	if err := f.show(s); err != nil {
		return quit, fmt.Errorf("[JSTACK-TUI] the terminal stopped answering: %w; rerun with --yes and --harness to skip the questions", err)
	}
	return s.outcome, nil
}

// harnessScreen is every run: the checkbox list of harnesses, found ones
// and saved picks checked. One Enter when nothing changed.
func (f *flow) harnessScreen(direction) (outcome, error) {
	choices, err := f.session.Harnesses()
	if err != nil {
		return quit, err
	}
	names := map[string]string{}
	options := make([]huh.Option[string], 0, len(choices))
	for _, choice := range choices {
		names[choice.Key] = choice.Name
		state := "not found"
		if choice.Found {
			state = "found"
		}
		options = append(options, huh.NewOption(fmt.Sprintf("%-11s %s, %s", choice.Name, choice.Where, state), choice.Key))
		if !f.harnessesInit && choice.Checked {
			f.harnesses = append(f.harnesses, choice.Key)
		}
	}
	f.harnessesInit = true
	field := huh.NewMultiSelect[string]().
		Title("Install into which harnesses?").
		Description("Space toggles, Enter continues.").
		Options(options...).
		Filterable(false).
		Value(&f.harnesses).
		Validate(func(picked []string) error {
			if len(picked) == 0 {
				return errors.New("pick at least one harness")
			}
			return nil
		})
	return f.ask(field, func() string {
		picked := make([]string, 0, len(f.harnesses))
		for _, key := range f.harnesses {
			picked = append(picked, names[key])
		}
		return "harnesses  " + strings.Join(picked, ", ")
	})
}

// skillRepoScreen asks once for a skills repo of the person's own.
func (f *flow) skillRepoScreen(direction) (outcome, error) {
	if f.session.SkillRepoAsked() {
		return skipped, nil
	}
	field := huh.NewInput().
		Title("Do you have a skills repo of your own?").
		Description("owner/name on GitHub, with a skills/ folder holding one folder per skill. Enter with nothing to skip.").
		Placeholder("owner/name").
		Value(&f.skillRepo).
		Validate(func(typed string) error {
			if strings.TrimSpace(typed) == "" {
				return nil
			}
			_, err := setup.RepoName(typed)
			return err
		})
	return f.ask(field, func() string {
		if strings.TrimSpace(f.skillRepo) == "" {
			return "skills repo  none"
		}
		name, _ := setup.RepoName(f.skillRepo)
		return "skills repo  " + name
	})
}

// repoNames is the repos this run installs from: the saved and flagged
// ones, plus the one typed.
func (f *flow) repoNames() ([]string, error) {
	names, err := f.session.SkillRepos()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(f.skillRepo) != "" {
		name, err := setup.RepoName(f.skillRepo)
		if err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, nil
}

// collisionScreens fetches the repos, then asks about each skill name in
// more than one source that no flag or saved pick settles.
func (f *flow) collisionScreens(entered direction) (outcome, error) {
	names, err := f.repoNames()
	if err != nil {
		return quit, err
	}
	open, err := f.session.Gather(f.ctx, names)
	if err != nil {
		return quit, err
	}
	if len(open) != len(f.collisions) {
		f.collisions = nil
		for _, collision := range open {
			f.collisions = append(f.collisions, collisionScratch{name: collision.Name, sources: collision.Sources})
		}
	}
	steps := make([]step, 0, len(f.collisions))
	for index := range f.collisions {
		steps = append(steps, f.collisionScreen(&f.collisions[index]))
	}
	result, err := sequence(steps...)(entered)
	if err != nil || result != next && result != skipped {
		return result, err
	}
	picks := map[string]string{}
	for _, collision := range f.collisions {
		picks[collision.name] = collision.pick
	}
	if err := f.session.PickSources(picks); err != nil {
		return quit, err
	}
	return result, nil
}

func (f *flow) collisionScreen(collision *collisionScratch) step {
	return func(direction) (outcome, error) {
		options := make([]huh.Option[string], 0, len(collision.sources)+1)
		for _, source := range collision.sources {
			options = append(options, huh.NewOption(setup.UseWording(source), source))
		}
		options = append(options, huh.NewOption("rename it yourself", setup.Rename))
		field := huh.NewSelect[string]().
			Title(fmt.Sprintf("Skill %q is in %s. Which one goes into the harnesses?", collision.name, strings.Join(collision.sources, " and "))).
			Options(options...).
			Value(&collision.pick)
		return f.ask(field, func() string {
			if collision.pick == setup.Rename {
				return collision.name + "  rename it yourself"
			}
			return collision.name + "  " + setup.UseWording(collision.pick)
		})
	}
}

// reposDirScreen asks once where the repos live: a folder found under
// home, or somewhere else, which the next screen takes typed.
func (f *flow) reposDirScreen(direction) (outcome, error) {
	if f.session.ReposDirsAsked() {
		return skipped, nil
	}
	guesses := f.session.ReposDirGuesses()
	if len(guesses) == 0 {
		f.reposPick = elsewhere
		return skipped, nil
	}
	if f.reposPick == "" {
		f.reposPick = guesses[0]
	}
	options := make([]huh.Option[string], 0, len(guesses)+1)
	for _, guess := range guesses {
		options = append(options, huh.NewOption(setup.Display(f.opts.Home, guess), guess))
	}
	options = append(options, huh.NewOption("somewhere else", elsewhere))
	field := huh.NewSelect[string]().
		Title("Where do your repos live?").
		Description("A folder with a git checkout in each subfolder. Setup reads each repo's Tracker line there.").
		Options(options...).
		Value(&f.reposPick)
	return f.ask(field, func() string {
		if f.reposPick == elsewhere {
			return "repos  somewhere else"
		}
		return "repos  " + setup.Display(f.opts.Home, f.reposPick)
	})
}

// reposTypedScreen takes the folder typed when no guess fit.
func (f *flow) reposTypedScreen(direction) (outcome, error) {
	if f.session.ReposDirsAsked() || f.reposPick != elsewhere {
		return skipped, nil
	}
	field := huh.NewInput().
		Title("Where do your repos live?").
		Description("A path, comma separated for more than one. Enter with nothing to skip.").
		Placeholder("~/code").
		Value(&f.reposTyped).
		Validate(func(typed string) error {
			_, err := setup.ParseReposDirs(typed, f.opts.Home)
			return err
		})
	return f.ask(field, func() string {
		dirs, _ := setup.ParseReposDirs(f.reposTyped, f.opts.Home)
		if len(dirs) == 0 {
			return "repos  none"
		}
		return "repos  " + shownList(f.opts.Home, dirs)
	})
}

// reposDirs is the folders this run scans: the saved and flagged ones,
// plus the answer.
func (f *flow) reposDirs() ([]string, bool, error) {
	dirs, err := f.session.ReposDirs()
	if err != nil {
		return nil, false, err
	}
	if f.session.ReposDirsAsked() {
		return dirs, true, nil
	}
	if f.reposPick == elsewhere {
		typed, err := setup.ParseReposDirs(f.reposTyped, f.opts.Home)
		if err != nil {
			return nil, false, err
		}
		return append(dirs, typed...), true, nil
	}
	return append(dirs, f.reposPick), true, nil
}

// trackerScreens asks about every repo that declares no tracker: which
// backend, what it needs, whether to open the PR, and whether the rest get
// the same answer.
func (f *flow) trackerScreens(entered direction) (outcome, error) {
	dirs, _, err := f.reposDirs()
	if err != nil {
		return quit, err
	}
	questions := f.session.Trackers(f.ctx, dirs)
	if !sameQuestions(f.trackers, questions) {
		f.trackers = nil
		for _, question := range questions {
			f.trackers = append(f.trackers, trackerScratch{question: question})
		}
	}
	var steps []step
	for index := range f.trackers {
		steps = append(steps, f.trackerScreensFor(index)...)
	}
	return sequence(steps...)(entered)
}

func sameQuestions(scratch []trackerScratch, questions []setup.TrackerQuestion) bool {
	if len(scratch) != len(questions) {
		return false
	}
	for index := range questions {
		if scratch[index].question != questions[index] {
			return false
		}
	}
	return true
}

// covered reports whether an earlier repo's answer was given for the rest.
func (f *flow) covered(index int) bool {
	for _, earlier := range f.trackers[:index] {
		if earlier.forRest {
			return true
		}
	}
	return false
}

func (f *flow) trackerScreensFor(index int) []step {
	current := &f.trackers[index]
	remaining := func() []string {
		var names []string
		for _, later := range f.trackers[index+1:] {
			names = append(names, later.question.Repo)
		}
		return names
	}
	backendOf := func() (setup.Backend, bool) {
		for _, entry := range setup.Backends() {
			if entry.Key == current.backend {
				return entry, true
			}
		}
		return setup.Backend{}, false
	}
	line := func() string {
		chosen, ok := backendOf()
		if !ok {
			return skip
		}
		argument := current.argument
		if argument == "" {
			argument = chosen.Default
		}
		return setup.TrackerLine(chosen, argument)
	}
	pick := func(direction) (outcome, error) {
		if f.covered(index) {
			return skipped, nil
		}
		options := []huh.Option[string]{huh.NewOption("skip this run", skip)}
		for _, entry := range setup.Backends() {
			options = append(options, huh.NewOption(entry.Label, entry.Key))
		}
		if current.backend == "" {
			current.backend = skip
		}
		field := huh.NewSelect[string]().
			Title(fmt.Sprintf("%s declares no tracker. Which one does it use?", current.question.Repo)).
			Description("The line goes into " + current.question.File + ".").
			Options(options...).
			Value(&current.backend)
		return f.ask(field, func() string {
			if chosen, ok := backendOf(); ok && chosen.Argument != "" {
				return current.question.Repo + "  " + chosen.Label
			}
			return current.question.Repo + "  " + line()
		})
	}
	argument := func(direction) (outcome, error) {
		chosen, ok := backendOf()
		if f.covered(index) || !ok || chosen.Argument == "" {
			return skipped, nil
		}
		description := "for example " + chosen.Example
		if chosen.Default != "" {
			description = "Enter for " + chosen.Default
		}
		field := huh.NewInput().
			Title(chosen.Label + " " + chosen.Argument).
			Description(description).
			Placeholder(chosen.Example).
			Value(&current.argument).
			Validate(func(typed string) error {
				if typed == "" && chosen.Default == "" || strings.ContainsAny(typed, " \t") {
					return fmt.Errorf("the %s is one word, for example %s", chosen.Argument, chosen.Example)
				}
				return nil
			})
		return f.ask(field, func() string { return current.question.Repo + "  " + line() })
	}
	openPR := func(direction) (outcome, error) {
		if f.covered(index) || current.backend == skip || !current.question.PROffer {
			return skipped, nil
		}
		field := huh.NewConfirm().
			Title(fmt.Sprintf("Open a PR for %s?", current.question.Repo)).
			Description("branch tracker-line, commit \"docs: name the tracker\", through gh").
			Affirmative("Yes").
			Negative("No, leave the line uncommitted").
			Value(&current.openPR)
		return f.ask(field, func() string {
			if current.openPR {
				return current.question.Repo + "  PR through gh"
			}
			return current.question.Repo + "  line left uncommitted"
		})
	}
	forRest := func(direction) (outcome, error) {
		rest := remaining()
		if f.covered(index) || len(rest) == 0 {
			return skipped, nil
		}
		field := huh.NewConfirm().
			Title(fmt.Sprintf("Same answer, %s, for the %d remaining?", line(), len(rest))).
			Description(strings.Join(rest, ", ")).
			Affirmative("Yes").
			Negative("No, ask about each").
			Value(&current.forRest)
		return f.ask(field, func() string {
			if current.forRest {
				return strings.Join(rest, ", ") + "  same answer"
			}
			return strings.Join(rest, ", ") + "  asked one by one"
		})
	}
	return []step{pick, argument, openPR, forRest}
}

// trackerAnswers is what the tracker screens decided, one per undeclared
// repo, the answer given for the rest carried down the list.
func (f *flow) trackerAnswers() []setup.TrackerAnswer {
	var answers []setup.TrackerAnswer
	var carried *trackerScratch
	for index := range f.trackers {
		current := &f.trackers[index]
		source := current
		if carried != nil {
			source = carried
		}
		answer := setup.TrackerAnswer{Dir: current.question.Dir, Repo: current.question.Repo, Skip: source.backend == skip || source.backend == ""}
		if !answer.Skip {
			for _, entry := range setup.Backends() {
				if entry.Key == source.backend {
					argument := source.argument
					if argument == "" {
						argument = entry.Default
					}
					answer.Line = setup.TrackerLine(entry, argument)
				}
			}
			answer.OpenPR = source.openPR && current.question.PROffer
		}
		answers = append(answers, answer)
		if carried == nil && current.forRest {
			carried = current
		}
	}
	return answers
}

// toolsScreen is the checkbox list of tools setup could install or update.
// Unchecked means skip.
func (f *flow) toolsScreen(direction) (outcome, error) {
	choices := f.session.Tools(f.ctx)
	var options []huh.Option[string]
	var states []string
	for _, choice := range choices {
		if !choice.Actionable {
			states = append(states, choice.State)
			continue
		}
		options = append(options, huh.NewOption(choice.Label, choice.Title))
		if !f.toolsInit && choice.Checked {
			f.tools = append(f.tools, choice.Title)
		}
	}
	f.toolsInit = true
	if len(options) == 0 {
		return skipped, nil
	}
	description := "Space toggles, Enter continues. Unchecked is skipped."
	if len(states) > 0 {
		description += "\n" + strings.Join(states, "\n")
	}
	field := huh.NewMultiSelect[string]().
		Title("Which tools?").
		Description(description).
		Options(options...).
		Filterable(false).
		Value(&f.tools)
	return f.ask(field, func() string {
		if len(f.tools) == 0 {
			return "tools  none"
		}
		var verbs []string
		for _, choice := range choices {
			if verb, _, ok := strings.Cut(choice.Label, ":"); ok && contains(f.tools, choice.Title) {
				verbs = append(verbs, verb)
			}
		}
		return "tools  " + strings.Join(verbs, ", ")
	})
}

// planScreen prints the plan and asks to apply it. With nothing to apply
// there is nothing to confirm, and the run goes straight to the report.
func (f *flow) planScreen(direction) (outcome, error) {
	answers, err := f.collect()
	if err != nil {
		return quit, err
	}
	plan, err := f.session.Plan(f.ctx, answers)
	if err != nil {
		return quit, err
	}
	f.plan, f.answers = plan, answers
	plan.Print(f.out, f.opts, answers)
	if !plan.Pending() {
		fmt.Fprintln(f.out, "\nEverything is in place. Nothing to apply.")
		f.confirmed = true
		return skipped, nil
	}
	f.confirmed = true
	field := huh.NewConfirm().
		Title("Apply?").
		Description("Everything above happens now.").
		Affirmative("Yes").
		Negative("No").
		Value(&f.confirmed)
	return f.ask(field, func() string {
		if f.confirmed {
			return "apply  yes"
		}
		return "apply  no"
	})
}

// collect is every answer in the values the plan takes.
func (f *flow) collect() (setup.Answers, error) {
	names, err := f.repoNames()
	if err != nil {
		return setup.Answers{}, err
	}
	dirs, asked, err := f.reposDirs()
	if err != nil {
		return setup.Answers{}, err
	}
	answers := setup.Answers{
		Harnesses:      f.harnesses,
		SkillRepos:     names,
		SkillRepoAsked: true,
		ReposDirs:      dirs,
		ReposDirsAsked: asked,
		Tools:          map[string]bool{},
		Trackers:       f.trackerAnswers(),
	}
	for _, title := range f.tools {
		answers.Tools[title] = true
	}
	return answers, nil
}

func contains(list []string, item string) bool {
	for _, candidate := range list {
		if candidate == item {
			return true
		}
	}
	return false
}

func shownList(home string, paths []string) string {
	parts := make([]string, 0, len(paths))
	for _, path := range paths {
		parts = append(parts, setup.Display(home, path))
	}
	return strings.Join(parts, ", ")
}
