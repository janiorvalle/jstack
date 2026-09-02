# Tools

jstack expects a few tools on the machine. Each one ships its own skill, which the tool maintains and updates. jstack doesn't copy those skills. It names the tool, says what it's for, and tells you how to get it.

If a tool is missing, run the check, show the install command, and install only when the human says yes or the session was already given permission to set things up. Then read the skill the tool installed and follow it.

The `sync` skill runs every check below at the end of an install and reports what's missing.

## git and gh

Version control and the PR host CLI. Every flow assumes both.

- Check: `command -v git && command -v gh && gh auth status`
- Install: `brew install git gh`, then `gh auth login`

## The work tracker

Where tasks live. Claim before you touch project files, record the files you expect to change, attach evidence at turn-in, and a human completes after the merge. That contract is the same whichever tracker the project uses. Read the installed tracker's skill for its commands.

**Quest** (current)
- Repo: https://github.com/janiorvalle/quest
- Check: `command -v quest`
- Install: see the repo README

**Linear** (coming)
- Not yet wired up. When it is, this section gets its check and its skill.

## roast

The independent code review gate. Reviews the current diff on a different model than the one that wrote it, and the flow loops until it says well done.

- Repo: https://github.com/janiorvalle/roast
- Check: `command -v roast`
- Install: see the repo README
- Skill: installs `roast` into your skills folder. Follow it for the scope fence and the loop.

## bgr

Turns a PR, commit, or diff into a review walkthrough. The HTML output is attached to the tracker as evidence once the diff is final.

- Repo: https://github.com/janiorvalle/better-git-review
- Check: `command -v bgr`
- Install: see the repo README
- Skill: installs `bgr` into your skills folder. Use `--format json --out <path>` from an agent, `--yes` on anything that might stage, never the interactive picker.

## Browser tooling

Whatever your harness gives you for driving a real browser. Needed to use a web UI the way a person would and to capture screenshots for evidence. Not a specific tool. Claude Code has a Chrome extension, Codex has a browser skill, others vary.

- Check: try to open a page. If you can't, say so before claiming a UI change is verified.
