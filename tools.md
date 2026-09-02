# Tools

jstack expects a few tools on the machine. Each one ships its own skill, which the tool maintains and updates. jstack doesn't copy those skills. It names the tool, says what it's for, and tells you how to get it.

If a tool is missing, run the check, show the install command, and install only when the human says yes or the session was already given permission to set things up. Then read the skill the tool installed and follow it.

`setup-jstack` runs every check below, installs each tool's own skill when the tool is present, and reports what's missing. The `Skill install` and `Skill folder` lines are what it parses.

## git and gh

Version control and the PR host CLI. Every flow assumes both.

- Check: `command -v git && command -v gh && gh auth status`
- Install: `brew install git gh`, then `gh auth login`

## The work tracker

Where tasks live. Claim before you touch project files, record the files you expect to change, attach evidence at turn-in, and a human completes after the merge. That contract is the same whichever tracker the project uses. Read the installed tracker's skill for its commands.

**Quest** (current)
- Repo: https://github.com/janiorvalle/quest
- Check: `command -v quest`
- Install: `curl -fsSL https://raw.githubusercontent.com/janiorvalle/quest/main/install.sh | sh`
- Skill install: `quest skill install`
- Skill folder: `quest`

**Linear** (coming)
- Not yet wired up. When it is, this section gets its check and its skill.

## roast

The independent code review gate. Reviews the current diff on a different model than the one that wrote it, and the flow loops until it says well done.

- Repo: https://github.com/janiorvalle/roast
- Check: `command -v roast`
- Install: `curl -fsSL https://raw.githubusercontent.com/janiorvalle/roast/main/install.sh | sh`
- Skill install: `roast install-skill --force`
- Skill folder: `roast`
- The skill covers the scope fence and the loop until well done. roast also installs it on its own the first time it runs in a repo.

## bgr

Turns a PR, commit, or diff into a review walkthrough. The HTML output is attached to the tracker as evidence once the diff is final.

- Repo: https://github.com/janiorvalle/better-git-review
- Check: `command -v bgr`
- Install: `curl -fsSL https://raw.githubusercontent.com/janiorvalle/better-git-review/main/install.sh | sh`
- Skill install: `bgr install-skill`
- Skill folder: `bgr`
- From an agent, use `--format json --out <path>`, pass `--yes` on anything that might stage, and never run the interactive picker or `bgr configure`.

## tokenomnom

Token usage and spend across your coding agents, plus transcript search. Not part of the flow, but the tools group ships it.

- Repo: https://github.com/janiorvalle/tokenomnom
- Check: `command -v tokenomnom`
- Install: `curl -fsSL https://raw.githubusercontent.com/janiorvalle/tokenomnom/main/install.sh | sh`
- Skill install: `tokenomnom install-skill`
- Skill folder: `tokenomnom`

## aws

The AWS CLI. `ecs-health` and anything else that talks to AWS goes through it with your existing profiles.

- Check: `command -v aws`
- Install: `brew install awscli`

## agent-browser

Drives a real browser from the command line. This is how an agent uses a web UI the way a person would and captures the screenshots the flow requires. Required, whichever harness you're in.

- Repo: https://github.com/vercel-labs/agent-browser
- Check: `command -v agent-browser`
- Install: `npm install -g agent-browser && agent-browser install`
- Skill install: `npx -y skills add vercel-labs/agent-browser --skill agent-browser -g -y -a claude-code -a codex --copy`
- Skill folder: `agent-browser`
- The skill lands in the harness skills folder and in `~/.agents/skills`, which Codex also reads.

Codex ships its own `playwright-interactive` skill for persistent browser and Electron sessions. It's Codex's, not part of this stack, and it stays wherever Codex put it.
