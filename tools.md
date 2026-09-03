# Tools

jstack expects a few tools on the machine. Each one ships its own skill, which the tool maintains and updates. jstack doesn't copy those skills. It names the tool, says what it's for, and tells you how to get it.

If a tool is missing, run the check, show the install command, and install only when the human says yes or the session was already given permission to set things up. Then read the skill the tool installed and follow it.

`jstack setup` runs every check below, installs each tool's own skill when the tool is present, and reports each tool as missing, outdated, or current. The `Check`, `Version`, `Repo`, `Install`, `Skill install`, and `Skill folder` lines are what it parses. `Version` prints the installed version; the latest comes from the GitHub releases of the `Repo` line, or from the npm registry for a tool installed with `npm install -g`. When that install line pins a version, `npm install -g name@1.2.3`, the pin is the latest: setup reports the tool outdated when it's behind the pin and ahead when it's past it, the update installs the pin either way, and the weekly vendor-bump workflow opens a PR when npm publishes a newer one. A section with a `Check` line and no `Install` line is a prerequisite: setup checks for it and points here when it's missing, but never installs or updates it. git and gh are the prerequisites.

## git and gh

Version control and the PR host CLI. Every flow assumes both. Setup checks for them and never installs them: the right command depends on the OS and its package manager, on Linux it needs sudo, and `gh auth login` is a conversation with GitHub that only you can have. Get them by hand, then rerun `jstack setup`.

- Check: `command -v git && command -v gh && gh auth status`
- macOS: `brew install git gh`
- Debian and Ubuntu: `sudo apt install git gh`
- Fedora: `sudo dnf install git gh`
- Anything else: https://github.com/cli/cli#installation covers gh, and git comes from the same package manager
- Then `gh auth login`

## The work tracker

Where tasks live. Claim before you touch project files, record the files you expect to change, attach evidence at turn-in, and a human completes after the merge. That contract is the same whichever tracker the project uses. Read the installed tracker's skill for its commands.

**Quest** (current)
- Repo: https://github.com/janiorvalle/quest
- Check: `command -v quest`
- Version: `quest --version`
- Install: `curl -fsSL https://raw.githubusercontent.com/janiorvalle/quest/main/install.sh | sh`
- Skill install: `quest skill install`
- Skill folder: `quest`

**Linear** (coming)
- Not yet wired up. When it is, this section gets its check and its skill.

## roast

The independent code review gate. Reviews the current diff on a different model than the one that wrote it, and the flow loops until it says well done.

- Repo: https://github.com/janiorvalle/roast
- Check: `command -v roast`
- Version: `roast --version`
- Install: `curl -fsSL https://raw.githubusercontent.com/janiorvalle/roast/main/install.sh | sh`
- Skill install: `roast install-skill --force`
- Skill folder: `roast`
- The skill covers the scope fence and the loop until well done. roast also installs it on its own the first time it runs in a repo.

## bgr

Turns a PR, commit, or diff into a review walkthrough. The HTML output is attached to the tracker as evidence once the diff is final.

- Repo: https://github.com/janiorvalle/better-git-review
- Check: `command -v bgr`
- Version: `bgr --version`
- Install: `curl -fsSL https://raw.githubusercontent.com/janiorvalle/better-git-review/main/install.sh | sh`
- Skill install: `bgr install-skill`
- Skill folder: `bgr`
- From an agent, use `--format json --out <path>`, pass `--yes` on anything that might stage, and never run the interactive picker or `bgr configure`.

## tokenomnom

Token usage and spend across your coding agents, plus transcript search. Not part of the flow, but the tools group ships it.

- Repo: https://github.com/janiorvalle/tokenomnom
- Check: `command -v tokenomnom`
- Version: `tokenomnom --version`
- Install: `curl -fsSL https://raw.githubusercontent.com/janiorvalle/tokenomnom/main/install.sh | sh`
- Skill install: `tokenomnom install-skill`
- Skill folder: `tokenomnom`

## agent-browser

Drives a real browser from the command line. This is how an agent uses a web UI the way a person would and captures the screenshots the flow requires. Required, whichever harness you're in.

- Repo: https://github.com/vercel-labs/agent-browser
- Check: `command -v agent-browser`
- Version: `agent-browser --version`
- Install: `npm install -g agent-browser@0.36.0 && agent-browser install`
- Its skill ships in this repo under `skills/agent-browser`, copied from upstream at the commit pinned in `vendor.json`, because jstack doesn't control the tool. `jstack setup` installs it like every other skill. That skill is a stub that runs `agent-browser skills get core`, so the instructions agents follow ship inside the CLI, which is why the install line pins the CLI version and `scripts/tool-bump.py` moves it through a PR.

Codex ships its own `playwright-interactive` skill for persistent browser and Electron sessions. It's Codex's, not part of this stack, and it stays wherever Codex put it.
