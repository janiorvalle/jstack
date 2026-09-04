# Decisions

Things we've decided that don't have a file to live in yet.

## For the mode

Written 2026-09-02, while drafting the principle skills. The mode gets written last.

- **Index every time, full file on demand.** The mode holds a one-line-per-principle list. The agent reads that list at the start of every multi-step task and opens a principle's full file only when it applies to the task in front of it.
- **Each principle file stands on its own.** The agent reads them one at a time, so small overlaps between files are fine. No cross-links that assume another file was read.
- **Mention a principle only when it changed a decision.** No list of every principle used at the end of a reply. Name one when it made the agent do something different from what it would have done anyway, and say what that was. Zero mentions on a small task is fine.
- **Generate the index from the description lines.** The list in the mode is built by a script from each skill's description, not typed by hand. One place to get the wording right, nothing to drift.

## Skills are harness-agnostic

Written 2026-09-02, while drafting the how skill.

Every skill has to work in Claude Code, Codex, Cursor, Pi, or anything else. No harness-specific tool names, agent types, model slugs, or config paths inside a skill. Say "spawn read-only subagents", "search the code", "use the browser tooling your harness gives you". The harness maps those to its own tools. Shell commands like grep are fine, since every harness has a shell.

## The skill index lives in the letter, not the mode

Written 2026-09-02, after the letter became the always-on copy.

The day-one decision put the one-line-per-skill index in the mode. That made sense before there was a letter. Now the letter is installed into every harness's instructions file and is in context on every turn, so it holds the index: principles as sections written by hand, workflows as a table generated from each skill's description by `scripts/build-index.py`. The mode no longer carries a copy, and no longer tells the agent to read one, since it's already there. One list, one place, generated.

## Third-party skills are committed, not fetched

Written 2026-09-03, while vendoring agent-browser and typescript-best-practices.

The first version fetched vendored skills at setup time, from upstream head or a pin nobody bumped, and the readme said they were never committed here. That meant a change to what our agents execute could land on every machine without anyone reading it. Now the rule is: a skill lives in this repo when jstack doesn't control the tool that owns it. Our own tools (quest, roast, bgr, tokenomnom) keep shipping their skill with the binary, since we review those repos already.

`vendor.json` stays the pin record. The committed copy is verbatim, license file alongside, and nobody edits it, because the weekly vendor-bump workflow copies the folder from upstream every run and opens a PR whenever the result differs from what's committed. A hand edit gets a PR putting the verbatim copy back. A skill's version is the last upstream commit that touched its folder, not the repo head, so an unrelated upstream commit doesn't open a PR. Vendored text is upstream's voice, not ours, so `verify.py` only checks that a SKILL.md exists and `build-index.py` leaves them out of the letter's table. `tools.md` still names agent-browser, since setup has to install the tool. That skill is a stub by upstream's design: it tells the agent to run `agent-browser skills get core`, and the real instructions ship inside the CLI at whatever version npm installed. The reviewed PR covers the stub, not the CLI's bundled text.

Bump PRs are opened with the workflow's own token, and GitHub doesn't start workflows for those, so CI doesn't run on them. The reviewer runs `make verify` before merging. A skill folder can't fail verify anyway, short of losing its SKILL.md.

## Upstream principle names stay dangling, our own skills answer to them

Written 2026-09-03, while deciding whether to vendor the two skills typescript-best-practices names.

The vendored `typescript-best-practices` skill opens with "apply the type-system-discipline principle skill first" and later points at boundary-discipline. Upstream, in `cursor/plugins`, those are `pstack/skills/principle-type-system-discipline` and `pstack/skills/principle-boundary-discipline`. Neither is vendored here, and they won't be.

jstack already has both. `strict-types` is type-system-discipline rewritten in our voice, pattern for pattern and test for test, plus the never-any and no-null rules from the letter. `validate-at-the-edges` is boundary-discipline the same way, plus the error contract. Vendoring the originals would put two copies of every rule into every harness, one in our voice and one in upstream's, and the two would drift the moment either side edits. The graph doesn't stop there either: type-system-discipline points at encode-lessons-in-structure, which we also cover as `dont-say-it-twice`, so each vendored principle would bring the next dangling name with it.

The rule: when a vendored skill names an upstream principle that jstack already states, the jstack skill carries the upstream name in its description line, and the reference stays as upstream wrote it. The description is what every harness shows in its skill list, so the name is visible before any file is opened. A note in the body says the same thing for an agent that gets there by search. `strict-types` answers to type-system-discipline. `validate-at-the-edges` answers to boundary-discipline. Nothing in the vendored text changes, so the weekly bump keeps working. A shim folder named after the upstream skill would resolve the name too, but it's a skill whose only job is pointing at the next skill, and two of them in every harness's list is clutter for a pointer.

Vendor an upstream principle only when jstack has no skill for it and doesn't want to write one.

## Setup is a binary with the skills inside

Written 2026-09-03, while replacing setup.py.

Onboarding needed a clone and a python path, and `/setup-jstack` was a silent no-op once installed, because the script looked for the checkout relative to itself and found `~/.claude` instead. Now `jstack` is a Go binary shaped like roast: one curl line, checksum verified, self-upgrade from GitHub releases. The skills, the letter, `tools.md`, and `vendor.json` are embedded at build time, so setup runs from any directory, and `go run ./cmd/jstack setup` from a checkout installs that checkout's files. No repo lookup, no environment variable.

Harnesses are detected by folder. The Claude Code and Codex rows first ask the variable each harness documents as moving that folder, `CLAUDE_CONFIG_DIR` and `CODEX_HOME`: when it is set and non-empty, detection, the skills, and the letter all go under the folder it names, and the plan prints that folder with the variable beside it. setup.py honored `CLAUDE_HOME`, which Claude Code's docs never mention; the binary honors the variable the harness reads, and OpenCode, Cursor, and Pi get no variable until their docs name one. The human picks from a numbered list with the found ones preselected. Picks live in `~/.jstack/config.json`, so reruns and `jstack upgrade` don't ask again. Backups of overwritten skills and replaced instructions files go to `~/.jstack/backup/<stamp>/<harness>/`, one place, instead of `.jstack-backup` folders inside skills folders and `.bak` files next to instructions files. Both backups are copies, not renames: with `CODEX_HOME` or `CLAUDE_CONFIG_DIR` on another mount, the skills folder and `~/.jstack` sit on different filesystems, and a rename between them fails with EXDEV. A changed skill is retired inside its own folder until the new one is in place, so nothing ever moves across the boundary. The git hooks step went with the script; `make install-hooks` in the checkout does that job.

Releases built for macOS and Linux only at first, because the installer was a shell script and every line in `tools.md` was POSIX shell. The Windows build came back with the PowerShell lines; that decision is below.

The table has five rows. Claude Code and Codex were verified on a real install. OpenCode's global skills folder is `~/.config/opencode/skills/` per its docs, plural, which corrected the second-hand `skill`. Pi's `~/.pi/agent/skills/` and `~/.pi/agent/AGENTS.md` match its docs. Cursor's `~/.cursor/skills` matches its docs, but `~/.cursor/rules/jstack.mdc` carries over from setup.py and Cursor's docs only describe project-level `.cursor/rules`, so that row is unverified. OpenCode, Cursor, and Pi also read `~/.claude/skills` or `~/.agents/skills`. Checked on OpenCode 1.18 with a fake home holding the same skill in both folders: it lists the skill once, keyed by name. Nothing to handle until a harness shows one twice.

## Setup knows outdated, not only missing

Written 2026-09-03, for quest 404.

Setup used to know two states per tool, missing or present, so a tool that was installed but behind stayed behind until someone noticed by hand. Now there are three: missing, outdated, and current, and the outdated offer has the same shape as the missing one, a y/N per tool with a terminal and `--update-tools` without one. The action is the tool's own install line, because every one of them is idempotent and installs the newest release; a separate update command per tool would be a second thing to keep right for no gain. After an update, setup reads the version again and reruns the tool's skill install even when a copy is already there, since the skill ships with the binary and the old copy describes the old binary.

`tools.md` carries a `Version` line per tool, the command that prints the installed version, in the same parsed format as `Check`. The latest version comes from the GitHub releases API through each section's `Repo` line, and from the npm registry when the install line is `npm install -g`, which today is only agent-browser. That is five requests per run, well inside the sixty an hour GitHub allows without a token, and they run at the same time with a five-second timeout so a dead network costs one wait, not five. A lookup that fails shows "latest unknown" and setup carries on: the network is never a reason not to install skills. git and gh have no `Version` line, because they come from the system package manager and updating them is not setup's job.

Versions are compared with `golang.org/x/mod/semver`, already a dependency for `jstack upgrade`, after adding the leading v the tools leave off when they print.

## git and gh are prerequisites, checked but never installed

Written 2026-09-03, for quest 410.

The git and gh section of `tools.md` installed with `brew install git gh`, so `jstack setup --install-tools` on a Linux box ran a command that doesn't exist there, then reported the check still failing. The quest offered two shapes: one install line per OS that the binary picks by `runtime.GOOS`, or git and gh as a manual step with only the curl-based tools auto-installed. The second one.

git and gh aren't tools the flow installs, they're what the flow stands on. Nothing gets cloned without git, and `gh auth login` is a conversation with GitHub that setup can't have for you. On Linux the right command depends on the distro, apt or dnf or pacman, and every one of them wants sudo; a setup tool running sudo on someone's box unasked is the wrong shape. An OS-switched install line would need a distro switch next, and that's the tell.

The rule: a `tools.md` section with a `Check` line and no `Install` line is a prerequisite. Setup checks for it, reports it as missing with a link to its own heading in `tools.md` on GitHub, and never offers to run anything for it, with or without a terminal. The section lists the ways to get it as prose, one line per platform. The curl-based tools and agent-browser keep their install lines, because those lines are the same on every OS the binary ships for.

## The agent-browser CLI is pinned

Written 2026-09-03, for quest 408.

The vendored `skills/agent-browser` is a stub by upstream's design: it tells the agent to run `agent-browser skills get core`, and the instructions the agent then follows ship inside the CLI. With `npm install -g agent-browser` unpinned, a new npm release changed what agents execute on every machine that ran setup, with no PR here. That was the one path left after the skills were vendored, so it closes the same way: the install line in `tools.md` is `npm install -g agent-browser@0.36.0`, setup treats the pin as the latest version and offers the install line whenever the machine has any other version, behind or ahead, since a machine ahead of the pin is running text nobody here read, and `scripts/tool-bump.py` runs in the weekly vendor-bump workflow to open a PR when npm publishes past it. The PR body links the npm version and the upstream release, and a human merges.

The pin lives in the install line, not in `vendor.json`, because the install line is what setup runs and what a person reads; a pin recorded somewhere else would have to be spliced in, and the two could drift. The bump is a second small script rather than a new entry kind in `vendor-bump.py`, because copying a folder from a tarball and rewriting one version in a markdown line share nothing but the PR ceremony, which is in the workflow.

The vendored `skills/agent-browser` folder and the pinned CLI move independently, each through its own PR. A skill bump and a CLI bump can land in either order; the stub only tells the agent to ask the CLI.

## Windows: the same binary, a PowerShell line beside each shell line

Written 2026-09-03, for quest 414.

The release now builds for Windows too, zipped like roast's, with `install.ps1` copied from quest's: download the zip, verify the SHA-256 against `checksums.txt`, smoke the binary, put `jstack.exe` in `%LOCALAPPDATA%\Programs\jstack`, add that folder to the user PATH, run `jstack setup`. The self-upgrade already had the Windows replacement dance from roast, backup beside the binary and a hidden cleanup command that waits for the old process to exit.

Setup runs each `tools.md` line in the shell the OS ships with, `sh -c` or `powershell -NoProfile -Command`, picked by `runtime.GOOS` in one function. The lines themselves take the per-OS shape: a plain line is the default, and a line suffixed with the OS name Go uses, `- Check (windows): \`Get-Command quest\``, overrides it on that OS. The parser resolves the suffix at parse time, so `Tool` and setup don't know there are two lines. The other shape, one line that works in both `sh` and PowerShell, was rejected: such a line is a trick nobody can read, and every tool already has a separate Windows install anyway. Only `Check` and `Install` needed a variant; `Version` and the skill lines are the same command in both shells and carry no suffix.

Only quest ships a PowerShell installer, so only quest gets a runnable `Install (windows)` line. roast, bgr, tokenomnom, and TruffleHog publish Windows archives with no installer, and a one-line PowerShell download-and-unzip with no checksum is the unreadable trick again. Their Windows install line is a sentence with no backticks, which the parser already treats as a step for a person: setup shows it and never runs it, the same as a prerequisite. When those repos ship an `install.ps1`, the line becomes `irm ... | iex` and setup installs them. Until then a Windows machine gets the skills, the letter, and quest from setup, and roast and bgr by hand.

The harness rows resolve on Windows without change. Go's `os.UserHomeDir` is `%USERPROFILE%` there, Claude Code's settings docs say `~/.claude` means `%USERPROFILE%\.claude` and `CLAUDE_CONFIG_DIR` moves it, and Codex's home-dir crate reads `CODEX_HOME` when set and non-empty and otherwise `~/.codex` through `dirs::home_dir`, which is the same folder. OpenCode, Cursor, and Pi keep their rows unverified on Windows. An instructions file a Windows editor saved with CRLF is planned and written back with LF, so the block matches on the next run instead of counting as changed forever.

Proof runs in CI on `windows-latest`: the Windows-tagged tests, then `scripts/install-smoke.ps1`, which builds and zips the binary, runs `install.ps1` twice, refuses a bad checksum, and runs setup into a throwaway profile folder where the `Get-Command` check for git and gh has to pass through PowerShell.

## Setup puts ~/.local/bin on PATH for the lines it runs

Written 2026-09-03, for quest 416.

Setup runs every `tools.md` line with `~/.local/bin` first on PATH when the folder is not on it yet. The jstack installer and every curl installer in `tools.md` put their tool there, and a fresh machine has the folder off PATH until the shell profile says so, so before this the check after an install failed and the error never said why. The prefix lives in `shellArguments`, next to the Windows line that refreshes the user PATH from the registry, so the Go side stays unaware of either. A PATH that already has the folder keeps its order, so a tool that sits earlier on PATH still wins and setup still reports it. The error for a check that fails after an install now names PATH and points at the folder the installer printed, since a tool that lands anywhere else is the case that remains. The other shape, an error that names `~/.local/bin` and leaves the run failing, was rejected: setup put the tool there, so setup finding it is what a person expects. So was putting the folder on PATH only for the check right after an install: the next run would report the same tool missing and install it again, and the skill line needs the tool on PATH too. The one thing the prefix hides is that the person's own terminal still can't see the folder, so setup says so on every run where `~/.local/bin` exists and is off PATH, with the profile line to add, and `install.sh` prints the same line.

## Windows installs the tools from setup

Written 2026-09-03, for quest 417.

The Windows section above left roast, bgr, tokenomnom, and TruffleHog as a sentence, a zip to unpack by hand, because only quest shipped a PowerShell installer. Now roast, better-git-review, and tokenomnom each carry an `install.ps1` in their own repo, the jstack one with the names swapped: download the release zip for the CPU, verify the SHA-256 against `checksums.txt`, run `--version` on what came out and require the number it prints to equal the release's, put the executables in `%LOCALAPPDATA%\Programs\<tool>`, add that folder to the user PATH. bgr and tokenomnom install both of their executables. Each repo's Windows CI job builds and zips the executables, runs the installer twice, and refuses a bad checksum, like the shell smoke on Linux. Their `Install (windows)` lines in `tools.md` are `irm ... | iex`, the same shape as quest's, so setup runs them.

TruffleHog is the odd one. Upstream ships no installer for Windows and the asset is a tar.gz next to a versioned checksums file. winget was checked first, since it ships with Windows and verifies checksums itself: `microsoft/winget-pkgs` has no TruffleHog manifest, the only publisher under `manifests/t` that starts with Truffle is Trufflesuite, which is Ethereum tooling. scoop has one, but scoop is a second package manager the person would install first, and the line would then depend on a bucket jstack doesn't control. So jstack owns the step. `scripts/install-trufflehog.ps1` is the same installer with tar.gz in place of zip, `tar -xzf` in place of `Expand-Archive` since Windows 10 ships bsdtar, and `trufflehog_<version>_checksums.txt` in place of `checksums.txt`. The script ships inside the binary, next to the skills and `tools.md`: setup writes it to `~/.jstack/scripts` before it runs any tool line, and the Windows line reads it from there into `iex`, the same shape as the other lines with a local file in place of a URL. The first cut pointed the line at the script's raw URL on main, and that had two problems: the URL exists only after the PR merges, so no branch could ever pass the Windows job, and a released binary would run whatever text main had that day instead of the script it was built with. Embedding fixes both, and the line needs no network for the script. Checksum verification is in every one of the five lines; a one-line download-and-extract with no checksum was rejected in the section above and stays rejected.

Every installer, jstack's own included, sends a GitHub token on the latest-release lookup when `<TOOL>_GITHUB_TOKEN`, `GH_TOKEN`, or `GITHUB_TOKEN` is set, the order quest's uses. Without one, GitHub allows sixty API requests an hour per address, and a CI runner shares its address, so the first Windows proof run saw 403s that the installers reported as no release found. The message now carries the HTTP status and, on a 403 or 429, names the variables and the version override that skips the lookup.

The parser still treats an `Install` line with no backticks as a step for a person. No tool uses that shape now, and the rule stays, because it is the right reading of such a line if one comes back.

The check after an install, `Get-Command roast`, runs in a PowerShell that setup starts, and the installer registered its folder in the user PATH, which the next terminal reads. Quest 416 closed that for both shells: the sh lines get `~/.local/bin` first on PATH, and the PowerShell lines get the user PATH read back from the registry before the line runs. So `setup --install-tools` on a fresh Windows box installs the tools and passes their checks in one run, with nothing on the PATH it started with, which is what the Windows CI smoke checks.

## Your own skills come from a repo you name

Written 2026-09-04, for quest 418.

A person's own skills lived only on the machine they wrote them on. Setup now asks once for a GitHub repo, `owner/name`, clones it into `~/.jstack/repos/<owner>/<name>`, fast-forwards it on every run, and installs its `skills/` folder through the same plan as jstack's own. Both the clone and the sync go through gh, `gh repo clone` and `gh repo sync --source owner/name`, run in the shell the `tools.md` lines use, so gh's login is what reaches GitHub and the binary never sees a token. Plain git has no credentials for a private clone until `gh auth setup-git` has run, and `--source` keeps a fork syncing from itself rather than its parent. Only `skills/` is read: no second letter, because two letters is the drift setup exists to prevent.

The plan is built from one list of sources, jstack's embedded folder first and then each repo, so a cloned `skills/` folder is a filesystem the same code plans and applies. Every skill in the plan carries its source, which is how the report names the repo after a skill and how "local, untouched" stays true for anything no source owns. One plan per source merged in setup was rejected: each source would list the others' skills as local.

A skill named the same in two sources is a collision, and setup stops on it rather than picking quietly: a terminal asks, keep jstack's, use the repo's, or rename; without one it refuses and prints the `--override name=source` line that answers. The pick is saved per skill in `~/.jstack/config.json`, printed on every later run, and dropped only on a run that reached every repo and found the name no longer collides. The question about a repo is asked once and remembered even when skipped, as an explicit `skill_repos_asked`, since obvious beats clever in a file people open.

Rules that keep a repo from doing harm: a sync that fails keeps the copy from the last run and a repo that can't be reached never hands its skills back to jstack; the clone is read through `os.OpenRoot`, so a symlink pointing outside it, `skills/` itself included, is refused; a folder a tool installs itself (`quest`, `roast`, `bgr`, `tokenomnom`) is left out, since the tool keeps that skill matched to its binary; a folder that isn't a lowercase name is left out and a local skill differing only in case stops the plan, since `Voice` and `voice` are one folder on the disks macOS and Windows ship with. A file the harness copy has and the source lacks is drift, hidden or not, except the few files the desktop drops into folders it browses, so a file a repo deletes leaves the harness on the next run. Forgetting a repo leaves its clone and its installed skills alone, except a skill that shadowed a jstack skill, which goes back to jstack's copy with a backup.

## An install line downloads first, so a failed download fails the line

Written 2026-09-04, for quest 422.

Every curl line in `tools.md` was `curl -fsSL <url> | sh`. When the download fails, curl exits non-zero but sh reads nothing and exits zero, and a pipeline's status is the last command's, so setup printed installed and only the check afterward said the tool was not there, with an error that told the person to look for a PATH problem. Roast found it on quest 416. Now each line is `script=$(mktemp) && curl -fsSL -o "$script" <url> && sh "$script"`: the download is its own step, and the line stops with curl's error when it fails. TruffleHog's line passes `-b ~/.local/bin` to the file instead of through `sh -s --`, since the script is an argument now. The Windows lines were already `irm ... | iex`, where a failed `irm` throws before `iex` runs, so they stay. `set -o pipefail` was the other shape and was rejected: `sh` on Linux is dash, which has no pipefail. The temp file is left for the OS to clean, since a trailing `rm` would either hide the install's exit status or add a step to every line for a file that is a few kilobytes in `$TMPDIR`. A test reads the real `tools.md` and fails on any curl line that pipes into sh again.

## Tool skills reach every picked harness

Written 2026-09-04, for quest 424.

quest, roast, bgr, and tokenomnom install their own skill into Claude Code's and Codex's folders, and a person who picked OpenCode or Pi never got those skills there, so every run found them missing and ran every skill install again. Setup now carries the result: after a tool's skill install line runs, it copies the folder the tool left behind into each picked harness that lacks it, from `~/.agents/skills` when the tool wrote there, else from the first harness in table order that has it, which is Claude Code's, picked or not. The tool's own install stays the source of truth: the copy is the tool's, no source owns the folder, so the skills plan reports it local and never touches it, and an update runs the tool's install line again and copies what it wrote.
