---
name: setup-jstack
description: "Use to put jstack on a machine or bring it up to date. Runs the jstack binary's setup: it finds the coding agents on the machine, shows the plan, asks which harnesses to install into, copies the skills, jstack's and the ones from a skills repo of your own, puts the letter in each instructions file, and offers the tools the flow needs. Reports what's still missing."
---

# Setup jstack

`jstack setup` is the whole thing. The binary carries the skills, the letter, and the tool list inside it, so it runs from any directory with no checkout on the machine.

## Get the binary

```sh
curl -fsSL https://raw.githubusercontent.com/janiorvalle/jstack/main/install.sh | sh
```

On Windows, in PowerShell:

```powershell
irm https://raw.githubusercontent.com/janiorvalle/jstack/main/install.ps1 | iex
```

The installer verifies the checksum, puts `jstack` in `~/.local/bin`, or `jstack.exe` in `%LOCALAPPDATA%\Programs\jstack` on your user PATH, and runs `jstack setup` once. When it's already installed, `jstack upgrade` fetches the newest release and reruns setup with the saved picks.

## What setup does, in order

1. **Plan.** Lists the harnesses it knows and which ones it found on the machine. Then the skills repos of your own, if any: each one is cloned or pulled with gh into `~/.jstack/repos/<owner>/<name>` and its `skills/` folder counted. For each picked harness: which skills are new, changed, or the same, with the repo named after each skill that comes from one, which local skills it will leave alone, and what happens to the instructions file. Then each tool from `tools.md`: missing with its install line, outdated with the installed and the latest version, or current. The latest version comes from the tool's GitHub releases, or from npm for agent-browser, one short request per tool; when that lookup fails the tool shows as current with "latest unknown" and setup carries on. git and gh are prerequisites: no version line, no install line, only ever missing or present, and a missing one shows a link to its section of `tools.md` instead of an install line.
2. **Ask.** With a terminal, once, whether you have a skills repo of your own, `owner/name`, Enter to skip; then a numbered list of harnesses with the found ones preselected, then a y/N per missing tool and per outdated tool. Without a terminal, it stops here, prints the exact rerun line, and changes nothing.
3. **Skills.** Copies the embedded skills and each repo's skills into each picked harness's skills folder. A skill that differs is moved to `~/.jstack/backup/<stamp>/<harness>/skills/` first. A skill no source owns is never touched.
4. **Letter.** Puts `AGENTS.md` into each picked harness's instructions file between `<!-- jstack:start -->` and `<!-- jstack:end -->`. A file with other content is replaced by the letter and backed up next to the skill backups, because this is an opinionated stack and two letters side by side is the drift it exists to prevent. `--keep-instructions` appends the block and leaves the file alone. Once the markers are in the file, later runs only change the text between them, so that choice holds without passing the flag again. The Cursor rule has to start with jstack's frontmatter, so a `jstack.mdc` without it is replaced and backed up even with the flag. The plan names any file that would be replaced.
5. **Tools.** Installs the missing tools and updates the outdated tools the human said yes to. An update runs the tool's own install line, which installs the newest release, then reads the version again. Then it runs each present tool's own skill install if its skill isn't there yet, and again for every tool it just updated, so the skill matches the binary. Nothing is installed without a yes or `--install-tools`, and nothing is updated without a yes or `--update-tools`.
6. **Picks.** Saves the harnesses, the skills repos, and the collision picks to `~/.jstack/config.json`. Later runs use them without asking. `--harness` overrides and replaces the harnesses.
7. **Report.** What was installed, updated, backed up, or left missing, and a reminder to restart the harness.

## Your own skills repo

Your own skills live only on the machine you wrote them on until something carries them. Setup does: name a GitHub repo, `owner/name`, with a `skills/` folder holding one folder per skill, each with a `SKILL.md`, and setup installs those skills beside jstack's in every picked harness, through the same plan: new, changed, same, backups, local skills untouched. Only `skills/` is read; nothing else in the repo matters to setup, and there is no second letter.

Setup asks once, with a terminal, and remembers the answer either way. Any time after that:

```sh
jstack setup --skill-repo owner/name          # add a repo, more than one is fine
jstack setup --forget-skill-repo owner/name   # stop installing from it; its skills stay in the harnesses as local skills
```

The clone lives in `~/.jstack/repos/<owner>/<name>` and is synced with `gh repo sync` on every run, so a skill you push shows up as changed on the next setup, with the old copy backed up. Both the clone and the sync go through gh, so a private repo needs nothing beyond `gh auth login`. A repo gh can't reach, private and not logged in, or a repo with no `skills/` folder, is reported with the reason and setup carries on with the rest. Check `gh auth status` for the private case.

A skill named the same in jstack and your repo, or in two of your repos, stops setup. With a terminal it asks: keep jstack's, use yours, or rename it yourself. The pick is saved and printed on every later run as `overridden by owner/name` or `kept from jstack`, so the copy that isn't installed never goes unnoticed. Without a terminal it refuses and prints the flag:

```sh
jstack setup --harness claude --yes --override land-pr=owner/name   # use yours
jstack setup --harness claude --yes --override land-pr=jstack       # keep jstack's
```

## From an agent

You have no terminal, so `jstack setup` prints the plan and changes nothing. Read it and bring the decision to the human in the fixed shape:

**Decide:** which harnesses jstack installs into.
**Options:** the harnesses the plan found, one line each with what changes there.
**Recommendation:** the found ones, unless the plan shows an instructions file with other content, in which case say so and offer `--keep-instructions`.

The plan ends with `add --skill-repo owner/name` until the human has answered the skills repo question once, so ask them in the same message whether they have one.

Then rerun with what they chose:

```sh
jstack setup --harness claude,codex --yes
jstack setup --harness all --yes --install-tools
jstack setup --harness all --yes --install-tools --update-tools
jstack setup --harness claude --yes --keep-instructions
jstack setup --harness claude,codex --yes --skill-repo owner/name
```

Read the report back to them. If a tool is still missing or outdated, quote its install line. If git or gh is missing, setup never installs them: send the human to the git and gh section of `tools.md` for their platform's command, then `gh auth login`, then rerun setup. If setup refused on a skill in two sources, bring that to the human as a decision too, with the two `--override` lines it printed as the options, and rerun with their pick. If the run changed anything, they need to restart the harness.

## Harnesses

| Key | Harness | Found by | Skills go to | Letter goes to | Moved by |
| --- | --- | --- | --- | --- | --- |
| `claude` | Claude Code | `~/.claude` | `~/.claude/skills` | `~/.claude/CLAUDE.md` | `CLAUDE_CONFIG_DIR` |
| `codex` | Codex | `~/.codex` | `~/.codex/skills` | `~/.codex/AGENTS.md` | `CODEX_HOME` |
| `opencode` | OpenCode | `~/.config/opencode` | `~/.config/opencode/skills` | `~/.config/opencode/AGENTS.md` | |
| `cursor` | Cursor | `~/.cursor` | `~/.cursor/skills` | `~/.cursor/rules/jstack.mdc`, with `alwaysApply` | |
| `pi` | Pi | `~/.pi/agent` | `~/.pi/agent/skills` | `~/.pi/agent/AGENTS.md` | |

On Windows `~` is `%USERPROFILE%`, which is where Claude Code and Codex read their folders from, so the rows resolve there without any change. When a row's variable is set and non-empty, that harness is found, and its skills and letter land, under the folder it names instead of the one under `~`. That is where the harness reads from, so setup follows it. The plan shows the folder with the variable next to it, `/work/codex (CODEX_HOME)`. An empty variable means the default.

OpenCode, Cursor, and Pi also read skills from Claude Code's folder or the shared `~/.agents/skills` folder. OpenCode was checked with the same skill in both folders and lists it once, keyed by name. Cursor and Pi are unverified. `decisions.md` has what was checked.

## Adding a vendored skill

One entry in `vendor.json`: repo, path inside it, pinned commit, license, and where the license file is. The pinned commit is the last upstream commit that touched the folder, not the repo head, or the first bump PR only moves the pin. Then `python3 scripts/vendor-bump.py restore <name>` copies the folder into `skills/` at that commit, license file alongside. The weekly vendor-bump workflow opens a PR whenever upstream moves. Never edit a vendored skill's text; the next bump would overwrite it.

## Adding a tool

One section in `tools.md` with a `Check` line, an `Install` line, and `Skill install` and `Skill folder` lines if the tool ships a skill. Leave the `Install` line out for a prerequisite setup should check but never install, and list how to get it as prose instead; setup then reports it missing with a link to that section. Add a `Version` line with the command that prints the installed version and a `Repo` line with the GitHub page when setup should offer updates; the latest version is read from that repo's releases, or from the npm registry when the install line is `npm install -g`. Pin the package, `npm install -g name@1.2.3`, when the tool carries text agents execute, as agent-browser does; setup then treats the pin as the latest, offers the install line to any machine not at the pin, and the weekly vendor-bump workflow opens a PR when npm moves past it. Every line is POSIX shell unless it carries an OS suffix, and setup runs it in `sh` on macOS and Linux and in Windows PowerShell on Windows. Give the tool a `Check (windows)` line, `Get-Command name`, since PowerShell has no `command -v`, and an `Install (windows)` line when the install differs, a PowerShell command in backticks, `irm https://.../install.ps1 | iex`. A tool whose upstream ships no PowerShell installer gets one in `scripts/` here, embedded in the binary: setup writes it to `~/.jstack/scripts` before it runs any tool line, and the line runs it from there, the way TruffleHog's does. A line with the suffix wins on that OS; the plain line is what every other OS gets. Windows lines use `;` between steps and single quotes. The binary parses those lines. Keep the format. The next release carries the new tool.

## Developing the binary

From a checkout, `make setup` runs `go run ./cmd/jstack setup` with that checkout's skills embedded. `make verify` is the gate.
