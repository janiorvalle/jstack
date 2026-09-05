# Tools

jstack expects a few tools on the machine. Each ships its own skill, which the tool maintains and updates. jstack doesn't copy those skills. It names the tool, says what it's for, and tells you how to get it.

If a tool is missing, run the check, show the install command, and install only when the human says yes or the session was already given permission to set things up. The install command is the `Install` line in this file, which `jstack setup` runs, never the one inside a tool's own skill: that one is upstream text and pulls whatever is newest, not the pin. Then read the skill the tool installed and follow it.

`jstack setup` runs every check below, installs each tool's own skill when the tool is present, and reports each tool as missing, outdated, or current. It parses the `Check`, `Version`, `Repo`, `Install`, `Skill install`, and `Skill folder` lines. `Version` prints the installed version. The latest comes from the GitHub releases of the `Repo` line, or from the npm registry for a tool installed with `npm install -g`. When that install line pins a version, `npm install -g name@1.2.3`, the pin is the latest: setup reports the tool outdated when it's behind the pin and ahead when it's past it, the update installs the pin either way, and the weekly vendor-bump workflow opens a PR when npm publishes a newer one. A section with a `Check` line and no `Install` line is a prerequisite: setup checks for it and points here when it's missing, but never installs or updates it. git and gh are the prerequisites.

Setup runs the lines in the shell the OS ships with: `sh` on macOS and Linux, Windows PowerShell on Windows. A line is POSIX shell unless it carries an OS suffix, and a suffixed line wins on that OS: `Check (windows)` runs on Windows, `Check` everywhere else. The suffix is the name Go gives the OS: `windows`, `darwin`, or `linux`. A line that's the same command in both shells, `roast --version`, carries no suffix. Windows lines use `;` between steps, since Windows PowerShell has no `&&`, and single quotes, since the line is passed through a double-quoted command line. An `Install` line with no backticks is a step for a person: setup shows it and never runs it. An `Install` line that fetches a script downloads it to a file first and runs the file only once the download succeeded, so a failed download fails the line. `curl | sh` runs nothing and exits zero when curl gets nothing, and setup would report the tool installed.

## git and gh

Version control and the PR host CLI. Every flow assumes both. Setup checks for them and never installs them: the right command depends on the OS and its package manager, on Linux it needs sudo, and `gh auth login` is a conversation with GitHub only you can have. Get them by hand, then rerun `jstack setup`.

- Check: `command -v git && command -v gh && gh auth status`
- Check (windows): `Get-Command git, gh -ErrorAction Stop; gh auth status`
- macOS: `brew install git gh`
- Debian and Ubuntu: `sudo apt install git gh`
- Fedora: `sudo dnf install git gh`
- Windows: `winget install --id Git.Git -e; winget install --id GitHub.cli -e`
- Anything else: https://github.com/cli/cli#installation covers gh, and git comes from the same package manager
- Then `gh auth login`

## roast

The independent code review gate. Reviews the current diff on a different model than the one that wrote it, and the flow loops until it says well done.

- Repo: https://github.com/janiorvalle/roast
- Check: `command -v roast`
- Check (windows): `Get-Command roast`
- Version: `roast --version`
- Install: `script=$(mktemp) && curl -fsSL -o "$script" https://raw.githubusercontent.com/janiorvalle/roast/main/install.sh && sh "$script"`
- Install (windows): `irm https://raw.githubusercontent.com/janiorvalle/roast/main/install.ps1 | iex`
- Skill install: `roast install-skill --force`
- Skill folder: `roast`
- The skill covers the scope fence and the loop until well done. roast also installs it on its own the first time it runs in a repo.
- roast refuses to run without TruffleHog on PATH, its `ROAST-SECRET-BINARY` error. roast's installer doesn't bring it, so setup installs it from the section below.

## TruffleHog

Secret scanner. roast runs it over the diff before review and refuses to start without it. Its official installer drops one binary into `~/.local/bin`, the folder the jstack installer already put on your PATH, on macOS and Linux with no sudo. Upstream ships no Windows installer, so jstack carries one inside the binary, `scripts/install-trufflehog.ps1`: setup writes it to `~/.jstack/scripts` before it runs any tool line, and the Windows install line runs it from there. It downloads the release tar.gz, verifies the SHA-256 against the release's checksums file, and puts `trufflehog.exe` in `%LOCALAPPDATA%\Programs\trufflehog` on your user PATH. It ships no skill.

- Repo: https://github.com/trufflesecurity/trufflehog
- Check: `command -v trufflehog`
- Check (windows): `Get-Command trufflehog`
- Version: `trufflehog --version`
- Install: `script=$(mktemp) && curl -fsSL -o "$script" https://raw.githubusercontent.com/trufflesecurity/trufflehog/main/scripts/install.sh && sh "$script" -b ~/.local/bin`
- Install (windows): `Get-Content -Raw (Join-Path $env:USERPROFILE '.jstack\scripts\install-trufflehog.ps1') | iex`

## bgr

Turns a PR, commit, or diff into a review walkthrough. The HTML output is attached to the tracker as evidence once the diff is final.

- Repo: https://github.com/janiorvalle/better-git-review
- Check: `command -v bgr`
- Check (windows): `Get-Command bgr`
- Version: `bgr --version`
- Install: `script=$(mktemp) && curl -fsSL -o "$script" https://raw.githubusercontent.com/janiorvalle/better-git-review/main/install.sh && sh "$script"`
- Install (windows): `irm https://raw.githubusercontent.com/janiorvalle/better-git-review/main/install.ps1 | iex`
- Skill install: `bgr install-skill`
- Skill folder: `bgr`
- From an agent, use `--format json --out <path>`, pass `--yes` on anything that might stage, and never run the interactive picker or `bgr configure`.

## tokenomnom

Token usage and spend across your coding agents, plus transcript search. Not part of the flow, but the tools group ships it.

- Repo: https://github.com/janiorvalle/tokenomnom
- Check: `command -v tokenomnom`
- Check (windows): `Get-Command tokenomnom`
- Version: `tokenomnom --version`
- Install: `script=$(mktemp) && curl -fsSL -o "$script" https://raw.githubusercontent.com/janiorvalle/tokenomnom/main/install.sh && sh "$script"`
- Install (windows): `irm https://raw.githubusercontent.com/janiorvalle/tokenomnom/main/install.ps1 | iex`
- Skill install: `tokenomnom install-skill`
- Skill folder: `tokenomnom`

## agent-browser

Drives a real browser from the command line. This is how an agent uses a web UI the way a person would and captures the screenshots the flow requires. Required, whichever harness you're in.

- Repo: https://github.com/vercel-labs/agent-browser
- Check: `command -v agent-browser`
- Check (windows): `Get-Command agent-browser`
- Version: `agent-browser --version`
- Install: `npm install -g agent-browser@0.36.0 && agent-browser install`
- Install (windows): `npm install -g agent-browser@0.36.0; agent-browser install`
- Its skill ships in this repo under `skills/agent-browser`, copied from upstream at the commit pinned in `vendor.json`, because jstack doesn't control the tool. `jstack setup` installs it like every other skill. The skill is a stub that runs `agent-browser skills get core`, so the instructions agents follow ship inside the CLI. That's why the install line pins the CLI version and `scripts/tool-bump.py` moves it through a PR.
- The stub also says `npm i -g agent-browser` when the tool is missing. That's upstream text and installs whatever npm has newest. Run the `Install` line above, or `jstack setup --install-tools`, and setup reports any other version as outdated or ahead.

Codex ships its own `playwright-interactive` skill for persistent browser and Electron sessions. It's Codex's, not part of this stack, and it stays wherever Codex put it.
