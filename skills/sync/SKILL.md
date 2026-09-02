---
name: sync
description: "Use to install jstack on a machine or bring an install up to date. Copies the skills into your harness's skills folder, backs up anything it overwrites, never touches skills it doesn't own, then checks that the tools jstack expects are installed."
---

# Sync

Put jstack on this machine, or update it. One command, then a report of what changed and what's missing.

## Defaults

- Source: the jstack checkout you're in, or `$JSTACK_REPO`.
- Targets: Codex at `${CODEX_HOME:-~/.codex}/skills`, Claude Code at `${CLAUDE_HOME:-~/.claude}/skills`. `--agent auto` picks the one you're running in. `--agent both` does both.
- Never delete a skill the repo doesn't own. Local-only skills stay.
- Back up any skill it overwrites, next to the destination, before writing.

## Flow

1. Resolve the source, the target, and the destination.
2. If the human wants the latest, `git status --short --branch` in the source. Clean, `git pull --ff-only`. Dirty, skip the pull and say which files.
3. Dry run. Show four groups: new, changed, same, local-only.
4. If there's anything new or changed, apply. Ask first unless the human already said to install or update.
5. Dry run again to confirm nothing is left to sync.
6. Run the tool checks from `tools.md` and list what's missing with its install line.
7. Tell the human to restart the harness so the skills load.

```bash
python3 skills/sync/scripts/sync.py --agent auto            # dry run
python3 skills/sync/scripts/sync.py --agent auto --apply
python3 skills/sync/scripts/sync.py --agent both --apply
python3 skills/sync/scripts/sync.py --skill voice --skill how --apply
```

## Report

Source path. Target and destination. Skills installed or updated. Backup location if anything was overwritten. Local-only skills left alone. Missing tools. Whether a restart is needed.
