---
name: setup-jstack
description: "Use to put jstack on a machine or bring it up to date. Runs the jstack binary's setup: it finds the coding agents on the machine, shows the plan, asks which harnesses to install into, copies the skills, jstack's and the ones from a skills repo of your own, puts the letter in each instructions file, and offers the tools the flow needs. Reports what's still missing."
---

# Setup jstack

`jstack setup` is the whole thing. The binary carries the skills, the letter, and the tool list, so it runs from any directory with no checkout.

## Get the binary

```sh
curl -fsSL https://raw.githubusercontent.com/janiorvalle/jstack/main/install.sh | sh
```

On Windows, in PowerShell:

```powershell
irm https://raw.githubusercontent.com/janiorvalle/jstack/main/install.ps1 | iex
```

The installer verifies the checksum, puts `jstack` in `~/.local/bin`, or `jstack.exe` in `%LOCALAPPDATA%\Programs\jstack` on your user PATH, and runs `jstack setup` once. `jstack upgrade` fetches the newest release and reruns setup with the saved picks.

## What setup does, in order

1. **Ask.** With a terminal, one screen per question, arrow keys and space to pick, Enter to continue, Esc to go back a screen, Ctrl-C to quit with nothing changed. Every run: the harnesses, a checkbox list with the found ones and the saved picks checked, so a rerun with nothing changed is one Enter. Once: a skills repo of your own, `owner/name`, Enter with nothing to skip. Once: where your repos live, a folder found under home or one typed, comma separated for more. Then, per repo that declares no tracker, the questions in step 6. Then the tools, a checkbox list of the missing and outdated ones with the line each would run; unchecked means skip. Without a terminal, setup prints the plan below, the exact rerun line, and changes nothing in the harnesses, though a saved skills repo is still synced.
2. **Plan.** A preview of steps 3 to 6, changing nothing: each skill as new, changed, the same, or local, with the repo it comes from, what happens to each instructions file, each tool as missing with its install line, outdated with the installed and latest version and the update line, or current, each repo's `Tracker:` line or `not declared`, and what the answers do to each undeclared one. Until a repos folder is named, that section lists the folders under home that hold checkouts. With a terminal the plan ends with a confirm, and nothing is written before it; when there is nothing to apply, there is nothing to confirm.
3. **Skills.** Copies the embedded skills and each repo's into each picked harness's skills folder. A skill that differs is moved to `~/.jstack/backup/<stamp>/<harness>/skills/` first. A skill no source owns is never touched.
4. **Letter.** Puts `AGENTS.md` between `<!-- jstack:start -->` and `<!-- jstack:end -->` in each picked harness's instructions file. A file with other content is replaced and backed up beside the skill backups, two letters side by side being the drift this stack exists to prevent, and the plan names any file it would replace. `--keep-instructions` appends the block instead. Once the markers are in, later runs change only the text between them, flag or not. A Cursor `jstack.mdc` not starting with jstack's frontmatter is replaced and backed up even with the flag. A `Tracker:` line in the letter stays out of the block: it names one repo's tracker, and every repo reads the instructions file.
5. **Tools.** Installs a missing tool only when checked or `--install-tools`, and updates an outdated one only when checked or `--update-tools`. The latest-version lookup is one request per tool; when it fails the tool shows current with "latest unknown". git and gh have no version or install line, only missing or present, with a link to their `tools.md` section. An update runs through whoever owns the binary, found with `command -v`: an npm install line updates with the pinned line wherever node lives; a binary under `brew --prefix` gets `brew upgrade <formula>`, the formula from a `Formula` line in `tools.md` when it isn't named like the binary; one in `~/.local/bin` gets the tool's own install line, which installs the newest release; one anywhere else is shown with its path and gets no offer, since the install line would drop a second copy that loses on PATH. Then setup reads the version again. Each present tool's skill install then runs where its skill is missing, and again after an update, so the skill matches the binary. Tools write their skill into Claude Code's and Codex's folders; setup copies it into a picked OpenCode, Cursor, or Pi that lacks it, from whichever of `~/.agents/skills`, Claude Code's, and Codex's folders the tool wrote last. That copy is the tool's, listed as local and left alone. One that no longer matches what the tool wrote counts as missing, so the install runs again and replaces it, old copy backed up.
6. **Repos.** Every git checkout one level down each repos folder, tracker read from `AGENTS.md` then `CLAUDE.md`, no network. With a terminal, per undeclared repo: a pick, skip, markdown tasks, GitHub Issues, Linear, or Jira, then what the backend needs, the folder, team key, or project key, then, when the repo can take it, whether to open the PR, then whether the remaining undeclared repos get the same answer. The line goes on its own line after the opening heading of `AGENTS.md`, or first without one, into `CLAUDE.md` when that's the only file, or into a new `AGENTS.md` alone when there's neither, after the confirm. The PR is offered given an origin, nothing else pending, and the default branch, the one `origin/HEAD` names, checked out at the remote's commit, read through git before the question and again before the write: branch `tracker-line`, commit `docs: name the tracker`, `gh pr create` with the ticket-shape body, back to the previous branch. Other uncommitted changes, a feature branch, or local commits ahead get the line and no offer. No `origin/HEAD` gets the `git remote set-head origin -a` line to run first. Until the PR merges the repo shows as waiting on `tracker-line` and isn't asked; delete the branch to be asked again. An `AGENTS.md` linking outside the repo is reported and left alone. git and gh run in the tool lines' shell, so gh's login reaches GitHub. Skip lasts one run, since the answer lives in the repo, not the config. Without a terminal nothing is written.
7. **Picks.** Saves the harnesses, skills repos, collision picks, and repos folders to `~/.jstack/config.json` for later runs. `--harness` replaces the harnesses. `--repos-dir <folder>` adds a repos folder and settles that question.
8. **Report.** Installed, updated, backed up, or left missing, and a reminder to restart the harness.

## Your own skills repo

A GitHub repo, `owner/name`, with a `skills/` folder, one folder per skill, each with a `SKILL.md`. Only `skills/` is read, and there is no second letter. Setup installs the skills beside jstack's through the same plan. After that:

```sh
jstack setup --skill-repo owner/name          # add a repo, more than one is fine
jstack setup --forget-skill-repo owner/name   # stop installing from it
jstack setup --no-skill-repo                  # there is none; setup stops asking and hinting
```

Forgetting a repo leaves its skills in the harnesses as local skills, untouched, except one that had replaced a jstack skill, which goes back to jstack's copy, yours backed up.

The clone in `~/.jstack/repos/<owner>/<name>` is synced with `gh repo sync` on every run, plan-only included, so a pushed skill shows as changed next time, old copy backed up. Both go through gh, so a private repo needs only `gh auth login`. A repo gh can't reach, or one with no `skills/` folder, is reported with the reason and setup carries on, and `gh auth status` covers the private case. A failed sync keeps the last copy and says so. A file symlinking outside the repo is refused. Left out of a repo and reported: a folder named after a tool's own skill, `roast`, `bgr`, or `tokenomnom`, since the tool installs that one and keeps it matched to its binary, and a folder that isn't lowercase, since `Voice` and `voice` are one folder on a Mac. A local skill differing from a source's only in case stops setup with the two names.

The same skill name in jstack and your repo, or in two of your repos, stops setup. With a terminal it asks on its own screen: keep jstack's, use yours, or rename it yourself. The pick is saved and printed on every later run as `overridden by owner/name` or `kept from jstack`. Without a terminal it refuses and prints the flag:

```sh
jstack setup --harness claude --yes --override land-pr=owner/name   # use yours
jstack setup --harness claude --yes --override land-pr=jstack       # keep jstack's
```

## From an agent

You have no terminal, so `jstack setup` prints the plan and changes nothing in the harnesses; a saved skills repo is still synced. Read it and bring the decision to the human in the fixed shape:

**Decide:** which harnesses jstack installs into.
**Options:** the harnesses the plan found, one line each with what changes there.
**Recommendation:** the found ones, unless the plan shows an instructions file with other content, in which case say so and offer `--keep-instructions`.

In the same message, ask about a skills repo while the plan ends with `add --skill-repo owner/name`, and where their repos live while the repos section says `add --repos-dir <folder>` with the folders it found. Pass `--skill-repo` or `--no-skill-repo`, and `--repos-dir`, on the rerun. Tracker questions need a terminal: for repos listed `not declared`, the human runs `jstack setup` themselves or you write the `Tracker:` line as the `tracker` skill describes.

Then rerun with what they chose:

```sh
jstack setup --harness claude,codex --yes
jstack setup --harness all --yes --install-tools
jstack setup --harness all --yes --install-tools --update-tools
jstack setup --harness claude --yes --keep-instructions
jstack setup --harness claude,codex --yes --skill-repo owner/name
jstack setup --harness claude,codex --yes --repos-dir ~/code
```

Read the report back. A tool still missing or outdated: quote its install line. git or gh missing: setup never installs them, so send the human to their section of `tools.md` for their platform's command, then `gh auth login`, then rerun. A skill refused for being in two sources: a decision with the two `--override` lines it printed as the options, then rerun with the pick. Anything changed: they restart the harness.

## Harnesses

| Key | Harness | Found by | Skills go to | Letter goes to | Moved by |
| --- | --- | --- | --- | --- | --- |
| `claude` | Claude Code | `~/.claude` | `~/.claude/skills` | `~/.claude/CLAUDE.md` | `CLAUDE_CONFIG_DIR` |
| `codex` | Codex | `~/.codex` | `~/.codex/skills` | `~/.codex/AGENTS.md` | `CODEX_HOME` |
| `opencode` | OpenCode | `~/.config/opencode` | `~/.config/opencode/skills` | `~/.config/opencode/AGENTS.md` | |
| `cursor` | Cursor | `~/.cursor` | `~/.cursor/skills` | `~/.cursor/rules/jstack.mdc`, with `alwaysApply` | |
| `pi` | Pi | `~/.pi/agent` | `~/.pi/agent/skills` | `~/.pi/agent/AGENTS.md` | |

On Windows `~` is `%USERPROFILE%`, where Claude Code and Codex read their folders from, so the rows resolve unchanged. A row's variable, set and non-empty, moves where that harness is found and where its skills and letter land; the plan shows it as `/work/codex (CODEX_HOME)`. An empty variable means the default.

OpenCode, Cursor, and Pi also read skills from Claude Code's folder or the shared `~/.agents/skills`. OpenCode was checked with the same skill in both and lists it once, keyed by name. Cursor and Pi are unverified. `decisions.md` has what was checked.

## Adding a vendored skill

One entry in `vendor.json`: repo, path inside it, pinned commit, license, and where the license file is. The pinned commit is the last upstream commit that touched the folder, not the repo head, or the first bump PR only moves the pin. Then `python3 scripts/vendor-bump.py restore <name>` copies the folder into `skills/` at that commit, license file alongside. The weekly vendor-bump workflow opens a PR whenever upstream moves. Never edit a vendored skill's text; the next bump would overwrite it.

## Adding a tool

One section in `tools.md`. The binary parses the lines, so keep the format. The next release carries the tool.

- `Check` and `Install`, plus `Skill install` and `Skill folder` when the tool ships a skill. A prerequisite setup checks but never installs gets no `Install` line and how to get it in prose; setup reports it missing with a link to the section.
- `Version`, the command printing the installed version, and `Repo`, the GitHub page, when setup should offer updates. The latest comes from that repo's releases, or from the npm registry when the install line is `npm install -g`. Pin it, `npm install -g name@1.2.3`, when the tool carries text agents execute, as agent-browser does: the pin is then the latest, offered to any machine not at it, and the weekly vendor-bump workflow opens a PR when npm moves past it.
- Lines are POSIX shell, run in `sh` on macOS and Linux, unless they carry an OS suffix. The suffixed line wins on that OS, in PowerShell on Windows, and the plain line serves every other OS. `Check (windows)` is `Get-Command name`, since PowerShell has no `command -v`. `Install (windows)`, when the install differs, is a PowerShell command in backticks, `irm https://.../install.ps1 | iex`. Windows lines use `;` between steps and single quotes.
- A tool whose upstream ships no PowerShell installer gets one in `scripts/` here, embedded in the binary. Setup writes it to `~/.jstack/scripts` before any tool line runs, and the line runs it from there, like TruffleHog's.

## Developing the binary

From a checkout, `make setup` runs `go run ./cmd/jstack setup` with that checkout's skills embedded. `make verify` is the gate.
