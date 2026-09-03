---
name: setup-jstack
description: "Use to put jstack on a machine or bring it up to date. Runs the jstack binary's setup: it finds the coding agents on the machine, shows the plan, asks which harnesses to install into, copies the skills, puts the letter in each instructions file, and offers the tools the flow needs. Reports what's still missing."
---

# Setup jstack

`jstack setup` is the whole thing. The binary carries the skills, the letter, and the tool list inside it, so it runs from any directory with no checkout on the machine.

## Get the binary

```sh
curl -fsSL https://raw.githubusercontent.com/janiorvalle/jstack/main/install.sh | sh
```

The installer verifies the checksum, puts `jstack` in `~/.local/bin`, and runs `jstack setup` once. When it's already installed, `jstack upgrade` fetches the newest release and reruns setup with the saved picks.

## What setup does, in order

1. **Plan.** Lists the harnesses it knows and which ones it found on the machine. For each picked harness: which skills are new, changed, or the same, which local skills it will leave alone, and what happens to the instructions file. Then each tool from `tools.md`: missing with its install line, outdated with the installed and the latest version, or current. The latest version comes from the tool's GitHub releases, or from npm for agent-browser, one short request per tool; when that lookup fails the tool shows as current with "latest unknown" and setup carries on. git and gh are prerequisites: no version line, no install line, only ever missing or present, and a missing one shows a link to its section of `tools.md` instead of an install line.
2. **Ask.** With a terminal, a numbered list of harnesses with the found ones preselected, then a y/N per missing tool and per outdated tool. Without a terminal, it stops here, prints the exact rerun line, and changes nothing.
3. **Skills.** Copies the embedded skills into each picked harness's skills folder. A skill that differs is moved to `~/.jstack/backup/<stamp>/<harness>/skills/` first. A skill jstack doesn't own is never touched.
4. **Letter.** Puts `AGENTS.md` into each picked harness's instructions file between `<!-- jstack:start -->` and `<!-- jstack:end -->`. A file with other content is replaced by the letter and backed up next to the skill backups, because this is an opinionated stack and two letters side by side is the drift it exists to prevent. `--keep-instructions` appends the block and leaves the file alone. Once the markers are in the file, later runs only change the text between them, so that choice holds without passing the flag again. The Cursor rule has to start with jstack's frontmatter, so a `jstack.mdc` without it is replaced and backed up even with the flag. The plan names any file that would be replaced.
5. **Tools.** Installs the missing tools and updates the outdated tools the human said yes to. An update runs the tool's own install line, which installs the newest release, then reads the version again. Then it runs each present tool's own skill install if its skill isn't there yet, and again for every tool it just updated, so the skill matches the binary. Nothing is installed without a yes or `--install-tools`, and nothing is updated without a yes or `--update-tools`.
6. **Picks.** Saves the harnesses to `~/.jstack/config.json`. Later runs use them without asking. `--harness` overrides and replaces them.
7. **Report.** What was installed, updated, backed up, or left missing, and a reminder to restart the harness.

## From an agent

You have no terminal, so `jstack setup` prints the plan and changes nothing. Read it and bring the decision to the human in the fixed shape:

**Decide:** which harnesses jstack installs into.
**Options:** the harnesses the plan found, one line each with what changes there.
**Recommendation:** the found ones, unless the plan shows an instructions file with other content, in which case say so and offer `--keep-instructions`.

Then rerun with what they chose:

```sh
jstack setup --harness claude,codex --yes
jstack setup --harness all --yes --install-tools
jstack setup --harness all --yes --install-tools --update-tools
jstack setup --harness claude --yes --keep-instructions
```

Read the report back to them. If a tool is still missing or outdated, quote its install line. If git or gh is missing, setup never installs them: send the human to the git and gh section of `tools.md` for their platform's command, then `gh auth login`, then rerun setup. If the run changed anything, they need to restart the harness.

## Harnesses

| Key | Harness | Found by | Skills go to | Letter goes to |
| --- | --- | --- | --- | --- |
| `claude` | Claude Code | `~/.claude` | `~/.claude/skills` | `~/.claude/CLAUDE.md` |
| `codex` | Codex | `~/.codex` | `~/.codex/skills` | `~/.codex/AGENTS.md` |
| `opencode` | OpenCode | `~/.config/opencode` | `~/.config/opencode/skills` | `~/.config/opencode/AGENTS.md` |
| `cursor` | Cursor | `~/.cursor` | `~/.cursor/skills` | `~/.cursor/rules/jstack.mdc`, with `alwaysApply` |
| `pi` | Pi | `~/.pi/agent` | `~/.pi/agent/skills` | `~/.pi/agent/AGENTS.md` |

OpenCode, Cursor, and Pi also read skills from Claude Code's folder or the shared `~/.agents/skills` folder. OpenCode was checked with the same skill in both folders and lists it once, keyed by name. Cursor and Pi are unverified. `decisions.md` has what was checked.

## Adding a vendored skill

One entry in `vendor.json`: repo, path inside it, pinned commit, license, and where the license file is. The pinned commit is the last upstream commit that touched the folder, not the repo head, or the first bump PR only moves the pin. Then `python3 scripts/vendor-bump.py restore <name>` copies the folder into `skills/` at that commit, license file alongside. The weekly vendor-bump workflow opens a PR whenever upstream moves. Never edit a vendored skill's text; the next bump would overwrite it.

## Adding a tool

One section in `tools.md` with a `Check` line, an `Install` line, and `Skill install` and `Skill folder` lines if the tool ships a skill. Leave the `Install` line out for a prerequisite setup should check but never install, and list how to get it as prose instead; setup then reports it missing with a link to that section. Add a `Version` line with the command that prints the installed version and a `Repo` line with the GitHub page when setup should offer updates; the latest version is read from that repo's releases, or from the npm registry when the install line is `npm install -g`. The binary parses those lines. Keep the format. The next release carries the new tool.

## Developing the binary

From a checkout, `make setup` runs `go run ./cmd/jstack setup` with that checkout's skills embedded. `make verify` is the gate.
