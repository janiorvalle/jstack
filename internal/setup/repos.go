package setup

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/janiorvalle/jstack/internal/prompt"
	"github.com/janiorvalle/jstack/internal/skills"
)

// skillRepo is one skills repo the human named, as found on this run. Its
// clone lives under ~/.jstack/repos/owner/name. verb says what happened to
// the clone; failure says why it can't be installed from this run, and a
// usable repo has none and carries its source. toolSkills are the folders
// in it that a tool installs itself, left out of the source.
type skillRepo struct {
	name       string
	dir        string
	verb       string
	failure    string
	source     skills.Source
	count      int
	toolSkills []string
}

func (r skillRepo) usable() bool {
	return r.failure == ""
}

// catalog is every place skills come from this run, jstack's embedded folder
// first and then each usable repo, and which source each name more than one
// of them holds is taken from.
type catalog struct {
	repos   []skillRepo
	sources []skills.Source
	picks   map[string]string
}

// repoName normalizes what a person types or pastes for a repo to owner/name.
func repoName(spec string) (string, error) {
	name := strings.TrimSpace(spec)
	name = strings.TrimPrefix(name, "https://")
	name = strings.TrimPrefix(name, "github.com/")
	name = strings.TrimSuffix(name, ".git")
	name = strings.Trim(name, "/")
	owner, repo, ok := strings.Cut(name, "/")
	if !ok || !plainName(owner) || !plainName(repo) {
		return "", fmt.Errorf("[JSTACK-SKILL-REPO] %q is not owner/name; expected a GitHub repo such as janiorvalle/work-skills, with a skills/ folder holding one folder per skill", spec)
	}
	return owner + "/" + repo, nil
}

func plainName(part string) bool {
	if part == "" {
		return false
	}
	for _, character := range part {
		switch {
		case character >= 'a' && character <= 'z', character >= 'A' && character <= 'Z', character >= '0' && character <= '9', character == '-', character == '_', character == '.':
		default:
			return false
		}
	}
	return true
}

// chooseRepos is the repos this run installs from: the saved ones, then the
// ones --skill-repo names, minus the ones --forget-skill-repo names.
func chooseRepos(saved []string, opts Options) ([]string, error) {
	forget := map[string]bool{}
	for _, spec := range opts.ForgetSkillRepos {
		name, err := repoName(spec)
		if err != nil {
			return nil, err
		}
		forget[name] = true
	}
	var repos []string
	seen := map[string]bool{}
	for _, spec := range append(append([]string{}, saved...), opts.SkillRepos...) {
		name, err := repoName(spec)
		if err != nil {
			return nil, err
		}
		if !seen[name] && !forget[name] {
			seen[name] = true
			repos = append(repos, name)
		}
	}
	return repos, nil
}

// askRepo asks once for a skills repo of the human's own. Enter skips.
func askRepo(ask *prompt.Prompt, out io.Writer) (string, error) {
	for {
		answer, err := ask.Ask("\nDo you have a skills repo of your own? owner/name, Enter to skip:")
		if err != nil {
			return "", err
		}
		if answer == "" {
			return "", nil
		}
		name, err := repoName(answer)
		if err == nil {
			return name, nil
		}
		fmt.Fprintln(out, err)
	}
}

// syncRepos clones each repo that isn't on the machine yet and pulls each
// one that is, both through gh so its login is what reaches GitHub: plain
// git has no credentials for a private clone until someone runs `gh auth
// setup-git`, and would stop at a username prompt. A pull that fails keeps
// the copy from the last run, so a dead network never changes the plan; a
// clone that fails is reported and left out, and setup carries on.
func syncRepos(ctx context.Context, opts Options, names []string) []skillRepo {
	repos := make([]skillRepo, 0, len(names))
	for _, name := range names {
		repos = append(repos, syncRepo(ctx, opts, name))
	}
	return repos
}

func syncRepo(ctx context.Context, opts Options, name string) skillRepo {
	repo := skillRepo{name: name, dir: filepath.Join(opts.Home, ".jstack", "repos", filepath.FromSlash(name))}
	command, verb := "gh repo clone "+name+" "+quote(runtime.GOOS, repo.dir), "cloned"
	if isDir(filepath.Join(repo.dir, ".git")) {
		command, verb = pullLine(runtime.GOOS, repo.dir), "pulled"
	} else if err := os.MkdirAll(filepath.Dir(repo.dir), 0o755); err != nil {
		repo.failure = fmt.Sprintf("cannot create %s: %v; make the home folder writable", display(opts.Home, filepath.Dir(repo.dir)), err)
		return repo
	}
	var output bytes.Buffer
	repo.verb = verb
	if err := opts.Shell(ctx, command, &output); err != nil {
		reason := fmt.Sprintf("`%s` failed: %v%s; if the repo is private, check `gh auth status`", command, err, lastLine(output.String()))
		if verb == "cloned" {
			repo.failure = reason
			return repo
		}
		repo.verb = "not pulled, using the copy from the last run: " + reason
	}
	skillsDir := filepath.Join(repo.dir, "skills")
	if !isDir(skillsDir) {
		repo.failure = "it has no skills/ folder; add one with a folder per skill, each with a SKILL.md, and push"
		return repo
	}
	// A repo can hold a symlink that points outside the clone. Reading
	// through a root refuses those, so nothing outside the repo is ever
	// copied into a harness.
	root, err := os.OpenRoot(skillsDir)
	if err != nil {
		repo.failure = fmt.Sprintf("cannot open %s: %v; make it readable, or delete the clone and rerun", display(opts.Home, skillsDir), err)
		return repo
	}
	repo.source = skills.Source{Name: name, Files: root.FS()}
	found, err := skills.Names(repo.source)
	if err != nil {
		repo.failure = err.Error()
		return repo
	}
	repo.count = len(found)
	return repo
}

// lastLine is the last thing gh or git printed, the line that says why.
func lastLine(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	last := strings.TrimSpace(lines[len(lines)-1])
	if last == "" {
		return ""
	}
	return ", " + last
}

// pullLine fast-forwards the clone from GitHub. gh has no flag for the
// folder to work in, so the line changes into it first, and stops there
// when it can't: a sync that ran in whatever folder setup was started from
// would move that repo instead.
func pullLine(operatingSystem, dir string) string {
	if operatingSystem == "windows" {
		return "Set-Location " + quote(operatingSystem, dir) + " -ErrorAction Stop; gh repo sync"
	}
	return "cd " + quote(operatingSystem, dir) + " && gh repo sync"
}

// quote makes a path one argument for the shell setup runs lines in: sh on
// macOS and Linux, PowerShell on Windows. Single quotes in both, with the
// quote character escaped the way each shell wants.
func quote(operatingSystem, path string) string {
	if operatingSystem == "windows" {
		return "'" + strings.ReplaceAll(path, "'", "''") + "'"
	}
	return "'" + strings.ReplaceAll(path, "'", `'\''`) + "'"
}

// buildCatalog lines the sources up, jstack first. A folder a tool installs
// itself, quest or roast say, is left out of every repo: a repo built from
// a whole skills folder carries those along, and the tool's own install
// keeps them matched to its binary.
func buildCatalog(embedded assets, repos []skillRepo) catalog {
	toolFolders := map[string]bool{}
	for _, tool := range embedded.tools {
		if tool.SkillFolder != "" {
			toolFolders[tool.SkillFolder] = true
		}
	}
	sources := []skills.Source{{Name: "jstack", Files: embedded.skills}}
	for index := range repos {
		repo := &repos[index]
		if !repo.usable() {
			continue
		}
		repo.toolSkills, repo.source.Files = leaveOutToolSkills(repo.source.Files, toolFolders)
		repo.count -= len(repo.toolSkills)
		sources = append(sources, repo.source)
	}
	return catalog{repos: repos, sources: sources}
}

func leaveOutToolSkills(files fs.FS, toolFolders map[string]bool) ([]string, fs.FS) {
	var found []string
	for folder := range toolFolders {
		if _, err := fs.Stat(files, folder+"/SKILL.md"); err == nil {
			found = append(found, folder)
		}
	}
	if len(found) == 0 {
		return nil, files
	}
	sort.Strings(found)
	return found, withoutFolders{FS: files, hidden: toolFolders}
}

// withoutFolders is a skills folder with some top-level folders hidden.
type withoutFolders struct {
	fs.FS
	hidden map[string]bool
}

func (w withoutFolders) Open(name string) (fs.File, error) {
	if w.hidden[strings.SplitN(name, "/", 2)[0]] {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return w.FS.Open(name)
}

func (w withoutFolders) ReadDir(name string) ([]fs.DirEntry, error) {
	if w.hidden[strings.SplitN(name, "/", 2)[0]] {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
	}
	entries, err := fs.ReadDir(w.FS, name)
	if name != "." || err != nil {
		return entries, err
	}
	kept := make([]fs.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if !w.hidden[entry.Name()] {
			kept = append(kept, entry)
		}
	}
	return kept, nil
}

func printRepos(out io.Writer, home string, repos []skillRepo) {
	if len(repos) == 0 {
		return
	}
	fmt.Fprintln(out, "\nskill repos")
	for _, repo := range repos {
		if !repo.usable() {
			fmt.Fprintf(out, "  %s  %s, FAILED: %s; setup carries on without it\n", repo.name, display(home, repo.dir), repo.failure)
			continue
		}
		fmt.Fprintf(out, "  %s  %s, %s, %d skills\n", repo.name, display(home, repo.dir), repo.verb, repo.count)
		for _, folder := range repo.toolSkills {
			fmt.Fprintf(out, "  %s  installed by the %s tool itself, the copy in %s is left out\n", folder, folder, repo.name)
		}
	}
}

// rememberOverrides is what the config keeps: this run's picks over the
// saved ones, minus any pick for a repo that is no longer named. A pick for
// a repo that failed to sync this run survives it.
func rememberOverrides(saved map[string]string, repoNames []string, picks map[string]string) map[string]string {
	known := map[string]bool{"jstack": true}
	for _, name := range repoNames {
		known[name] = true
	}
	kept := map[string]string{}
	for name, source := range saved {
		if known[source] {
			kept[name] = source
		}
	}
	for name, source := range picks {
		kept[name] = source
	}
	return kept
}

// printOverrides says where each colliding skill name comes from on every
// run, so a repo copy that isn't installed never goes unnoticed.
func printOverrides(out io.Writer, collisions []skills.Collision, picks map[string]string) {
	for _, collision := range collisions {
		pick := picks[collision.Name]
		verb := "overridden by"
		if pick == "jstack" {
			verb = "kept from"
		}
		fmt.Fprintf(out, "  %s  %s %s, not installed from %s\n", collision.Name, verb, pick, strings.Join(without(collision.Sources, pick), ", "))
	}
}

// resolveCollisions picks a source for every name more than one source
// holds: a --override flag first, then the saved pick, then the human when
// there is a terminal. Without one the refusal names the flag that picks.
func resolveCollisions(ask *prompt.Prompt, collisions []skills.Collision, saved, flags map[string]string) (map[string]string, error) {
	picks := map[string]string{}
	byName := map[string]skills.Collision{}
	for _, collision := range collisions {
		byName[collision.Name] = collision
	}
	for name, source := range flags {
		collision, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("[JSTACK-OVERRIDE] skill %q is not in more than one source, so there is nothing to override; %s", name, collisionList(collisions))
		}
		if !holds(collision, source) {
			return nil, fmt.Errorf("[JSTACK-OVERRIDE] skill %q is not in %s; it is in %s; example: --override %s=%s", name, source, strings.Join(collision.Sources, " and "), name, collision.Sources[1])
		}
		picks[name] = source
	}
	for _, collision := range collisions {
		if _, done := picks[collision.Name]; done {
			continue
		}
		if source, ok := saved[collision.Name]; ok && holds(collision, source) {
			picks[collision.Name] = source
			continue
		}
		if ask == nil {
			return nil, refusal(collision)
		}
		source, err := askCollision(ask, collision)
		if err != nil {
			return nil, err
		}
		picks[collision.Name] = source
	}
	return picks, nil
}

func collisionList(collisions []skills.Collision) string {
	if len(collisions) == 0 {
		return "no skill is in more than one source"
	}
	parts := make([]string, 0, len(collisions))
	for _, collision := range collisions {
		parts = append(parts, collision.Name+" ("+strings.Join(collision.Sources, ", ")+")")
	}
	return "the skills in more than one source: " + strings.Join(parts, "; ")
}

func refusal(collision skills.Collision) error {
	flags := make([]string, 0, len(collision.Sources))
	for _, source := range collision.Sources {
		flags = append(flags, fmt.Sprintf("--override %s=%s to %s", collision.Name, source, useWording(source)))
	}
	return fmt.Errorf("[JSTACK-SKILL-COLLISION] skill %q is in %s, and there is no terminal to ask which one goes into the harnesses; rerun with %s, or rename the folder in %s", collision.Name, strings.Join(collision.Sources, " and "), strings.Join(flags, ", "), strings.Join(without(collision.Sources, "jstack"), " or "))
}

func askCollision(ask *prompt.Prompt, collision skills.Collision) (string, error) {
	labels := make([]string, 0, len(collision.Sources)+1)
	for _, source := range collision.Sources {
		labels = append(labels, useWording(source))
	}
	labels = append(labels, "rename it yourself")
	index, err := ask.Choose(fmt.Sprintf("Skill %q is in %s. Which one goes into the harnesses?", collision.Name, strings.Join(collision.Sources, " and ")), labels)
	if err != nil {
		return "", err
	}
	if index == len(collision.Sources) {
		return "", fmt.Errorf("[JSTACK-SKILL-COLLISION] setup stopped so you can rename the %q folder in %s; push, then rerun jstack setup. The harnesses are unchanged", collision.Name, strings.Join(without(collision.Sources, "jstack"), " or "))
	}
	return collision.Sources[index], nil
}

func useWording(source string) string {
	if source == "jstack" {
		return "keep jstack's"
	}
	return "use " + source + "'s"
}

func holds(collision skills.Collision, source string) bool {
	for _, candidate := range collision.Sources {
		if candidate == source {
			return true
		}
	}
	return false
}

func without(sources []string, skip string) []string {
	rest := make([]string, 0, len(sources))
	for _, source := range sources {
		if source != skip {
			rest = append(rest, source)
		}
	}
	return rest
}
