---
name: setup-jstack
description: "Use to put jstack on a machine or bring it up to date. Installs the skills into your harness's skills folder, including the vendored third-party ones, checks every tool the flow expects and offers to install what's missing, installs each tool's own skill, and reports what's still not there."
---

# Setup jstack

One command on a fresh machine. Everything the flow needs ends up installed, or you get a list of what's still missing and how to get it.

## What it does, in order

1. **Skills.** Copies `skills/` into the harness's skills folder. Backs up anything it overwrites. Never touches a skill it doesn't own. The vendored third-party skills are in `skills/` too, so nothing is fetched from the network.
2. **Instructions.** Makes the harness's user-level instructions file the letter. `~/.claude/CLAUDE.md`, `~/.codex/AGENTS.md`, or `~/.cursor/rules/jstack.mdc` with `alwaysApply` set. The letter goes in as a block between `<!-- jstack:start -->` and `<!-- jstack:end -->`. If the file already has other content, it's replaced and the old file is backed up next to it as `.bak-<timestamp>`, because this is an opinionated stack and two letters side by side is the drift it exists to prevent. Pass `--keep-instructions` to append the block and leave your file alone instead. Later runs only change the text between the markers. This is what makes the mode always on, since the harnesses have no plugin hook of their own.
3. **Tools.** For each tool in `tools.md`, runs its check. Missing tools get listed with their install line. Nothing gets installed unless you say so, or the session was already given permission to set things up.
4. **Tool skills.** For each tool that's present, runs its skill install command if the skill isn't in the folder yet. The tool owns that skill and keeps it current.
5. **Hooks.** If you're in the jstack checkout, points git at `.githooks/`.
6. **Report.** What changed, what was backed up, what's missing, and a reminder to restart the harness.

## Defaults

- Source: the jstack checkout you're in, or `$JSTACK_REPO`.
- Target: `--agent auto` picks the harness you're running in. `--agent both` does Codex and Claude Code. `--agent all` adds Cursor. `--agent cursor` on its own works too.
- Dry run unless `--apply`.
- Tools are never installed unless `--install-tools` is passed. Ask the human first.
- An existing instructions file is replaced and backed up unless `--keep-instructions` is passed.

```bash
python3 skills/setup-jstack/scripts/setup.py --agent auto                    # dry run, shows everything it would do
python3 skills/setup-jstack/scripts/setup.py --agent both --apply
python3 skills/setup-jstack/scripts/setup.py --agent auto --apply --install-tools
python3 skills/setup-jstack/scripts/setup.py --skill voice --skill how --apply
```

## Adding a vendored skill

One entry in `vendor.json`: repo, path inside it, pinned commit, license, and where the license file is. The pinned commit is the last upstream commit that touched the folder, not the repo head, or the first bump PR only moves the pin. Then `python3 scripts/vendor-bump.py restore <name>` copies the folder into `skills/` at that commit, license file alongside. The weekly vendor-bump workflow opens a PR whenever upstream moves. Never edit a vendored skill's text; the next bump would overwrite it.

## Adding a tool

One section in `tools.md` with a check line, an install line, and a skill install line if the tool ships a skill. The script parses those lines. Keep the format.

## Report

Source, target and destination, skills installed or updated, backup location, tools missing with install lines, tool skills installed, whether a restart is needed.
