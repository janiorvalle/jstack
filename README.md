# jstack

<p align="center">
  <img src="assets/hero.png" alt="jstack. how I work, stacked." width="840">
</p>

How I work with coding agents, written down as skills.

One flow from claiming a task to turning it in with proof. Twenty-four principles, each one rule with a test. Twenty-six workflows for the parts that need steps. A mode skill that ties it together. Every file is written for an agent in Claude Code, Codex, Cursor, or anything else that loads skills from a folder.

The opinions are the point. A human stays in the merge seat. Every claim ships with evidence. The obvious solution beats the clever one. If you don't agree, this isn't your stack, and that's fine.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/janiorvalle/jstack/main/install.sh | sh
```

That puts the `jstack` binary in `~/.local/bin`, checksum verified, and runs `jstack setup`. Setup finds the coding agents on the machine, shows what it would do, and asks which to install into. Then it copies the skills, puts the letter in each harness's instructions file, and offers the tools the flow needs. It never touches a skill it doesn't own, and it backs up everything it overwrites under `~/.jstack/backup/`.

On Windows, in PowerShell:

```powershell
irm https://raw.githubusercontent.com/janiorvalle/jstack/main/install.ps1 | iex
```

That puts `jstack.exe` in `%LOCALAPPDATA%\Programs\jstack` on your user PATH and runs `jstack setup`, with the tool checks and installs in Windows PowerShell.

Run `jstack setup` again any time. It remembers the harnesses you picked. `jstack upgrade` fetches the newest release and reruns setup.

Setup asks two more questions, once each. A skills repo of your own, `owner/name` on GitHub with a `skills/` folder: it's cloned with gh under `~/.jstack/repos`, pulled on every run, and its skills go beside jstack's in every harness you picked. And where your repos live, guessed from the folders under home that hold git checkouts: `~/code`, `~/github`, `~/src`, `~/projects`, `~/dev`. Setup lists every repo there with its `Tracker:` line or `not declared` and, with a terminal, asks about each undeclared one: one of the four backends the `tracker` skill knows, or skip, same for all if you like. Then it writes the line into the repo's `AGENTS.md` and offers a one-line PR through gh.

Without a terminal, setup prints the plan and the exact flags to apply it, and changes nothing:

```sh
jstack setup --harness claude,codex --yes   # or --harness all
```

- `--install-tools` installs the missing tools without asking. `--update-tools` updates the outdated ones.
- `--keep-instructions` appends the letter to an instructions file that has other content instead of replacing it.
- `--skill-repo owner/name` adds a skills repo. `--forget-skill-repo` drops one.
- `--override name=owner/name` picks between two skills with the same name, which otherwise stop setup and ask.
- `--repos-dir <folder>` answers the repos question.

## Use it

Restart your harness so the skills load. Then start any multi-step task with `/jstack-mode`, or say "work the jstack way".

## Tools

The flow leans on three tools. `jstack setup` offers each one that's missing or behind its latest release.

- **roast**, the independent review gate. A different model reviews the diff until it says well done. It needs TruffleHog for its secret scan, and setup installs that too.
- **bgr**, the review walkthrough. Its HTML is the last piece of evidence before turn-in.
- **agent-browser**, a real browser from the command line, for using what you built like a person would.

The work tracker isn't a tool setup installs. Each repo names its own on a `Tracker:` line in its instructions file, and the `tracker` skill has the contract, the ticket shape, and the commands for markdown tasks, GitHub Issues, Linear, and Jira.

Each tool ships its own skill and installs it. `tools.md` has the check, version, and install lines setup runs, with `~/.local/bin` on PATH so a tool just installed there is found in the same run. git and gh are prerequisites: setup checks for them and says where to get them, never installs them. The right command depends on the OS, and `gh auth login` is yours to run.

## What's in it

- `AGENTS.md` is the letter: who's who, what we're doing, and the rules that matter first.
- `skills/jstack-mode/` is the front door: the flow, the rules, and an index of every skill, generated from their description lines.
- `skills/*/` with `kind: principle` in the frontmatter are the principles. One rule each.
- The rest are workflows: `how`, `why`, `architect`, `arena`, `swarm`, `land-pr`, `worktree`, and so on.
- `tools.md` names the tools the flow expects and how to get them. A `(windows)` line runs there in PowerShell, the plain line is POSIX shell for macOS and Linux. The agent-browser line pins the CLI version because its skill text ships inside the CLI, and `scripts/tool-bump.py` moves that pin through a weekly PR.
- `vendor.json` pins the third-party skills in `skills/`, the ones whose tool jstack doesn't control, so their text changes through a reviewed PR. `scripts/vendor-bump.py` copies each in at its pin, and a weekly workflow opens a bump PR when upstream moves. Our own tools, roast, bgr, and tokenomnom, ship their skill with the binary instead.
- `cmd/jstack` and `internal/` are the binary, with the skills, the letter, `tools.md`, and `vendor.json` embedded at build time, so setup runs from anywhere.
- `tasks/` is this repo's own tracker, markdown tasks in the `tracker` skill's shape, named by the `Tracker:` line at the top of the letter. Setup leaves that line out of what it installs.
- `decisions.md` records the choices made while building this, so nobody relitigates them.

## Development

```sh
make install-hooks   # gitleaks plus the verify script on every commit
make verify          # what CI runs: gofmt, build, vet, test, then the skill checks
make index           # rebuild the letter's workflows table after changing a description
make setup           # go run ./cmd/jstack setup, with this checkout's skills embedded
```

See `CONTRIBUTING.md` for the shape of a skill and the CLA.

## Credit

The idea of a personal agent stack with a mode as the front door, small principle files, and a fan-out pattern for understanding code comes from [pstack](https://github.com/cursor/plugins/tree/main/pstack). Everything here is written from scratch.

## License

MIT.
