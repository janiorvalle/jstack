---
name: setup-jstack
description: "Use to put jstack on a machine or bring it up to date. Installs the skills into your harness's skills folder, fetches the vendored third-party skills at their pinned versions, checks every tool the flow expects and offers to install what's missing, installs each tool's own skill, and reports what's still not there."
---

# Setup jstack

One command on a fresh machine. Everything the flow needs ends up installed, or you get a list of what's still missing and how to get it.

## What it does, in order

1. **Skills.** Copies `skills/` into the harness's skills folder. Backs up anything it overwrites. Never touches a skill it doesn't own.
2. **Vendor skills.** Fetches each entry in `vendor.json` from its repo at its pinned commit and installs it the same way. These are third-party skills the stack depends on but doesn't rewrite.
3. **Instructions.** Puts `AGENTS.md` from the repo into the harness's user-level instructions file, `~/.claude/CLAUDE.md`, `~/.codex/AGENTS.md`, or `~/.cursor/rules/jstack.mdc` with `alwaysApply` set, as a block between `<!-- jstack:start -->` and `<!-- jstack:end -->`. Later runs replace the block and never touch anything outside it. This is what makes the mode always on, since the harnesses have no plugin hook of their own.
4. **Tools.** For each tool in `tools.md`, runs its check. Missing tools get listed with their install line. Nothing gets installed unless you say so, or the session was already given permission to set things up.
5. **Tool skills.** For each tool that's present, runs its skill install command if the skill isn't in the folder yet. The tool owns that skill and keeps it current.
6. **Hooks.** If you're in the jstack checkout, points git at `.githooks/`.
7. **Report.** What changed, what was backed up, what's missing, and a reminder to restart the harness.

## Defaults

- Source: the jstack checkout you're in, or `$JSTACK_REPO`.
- Target: `--agent auto` picks the harness you're running in. `--agent both` does Codex and Claude Code. `--agent all` adds Cursor. `--agent cursor` on its own works too.
- Dry run unless `--apply`.
- Tools are never installed unless `--install-tools` is passed. Ask the human first.

```bash
python3 skills/setup-jstack/scripts/setup.py --agent auto                    # dry run, shows everything it would do
python3 skills/setup-jstack/scripts/setup.py --agent both --apply
python3 skills/setup-jstack/scripts/setup.py --agent auto --apply --install-tools
python3 skills/setup-jstack/scripts/setup.py --skill voice --skill how --apply
```

## Adding a vendored skill

One entry in `vendor.json`. Repo, path inside it, pinned commit, license, which harnesses. Bumping the commit is how you take an update. Nothing from a vendored skill is ever committed to this repo.

## Adding a tool

One section in `tools.md` with a check line, an install line, and a skill install line if the tool ships a skill. The script parses those lines. Keep the format.

## Report

Source, target and destination, skills installed or updated, vendor skills fetched, backup location, tools missing with install lines, tool skills installed, whether a restart is needed.
