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

Harnesses are detected by folder, never by environment variable, so a `CODEX_HOME` that points elsewhere is not honored yet; that is quest 409. The human picks from a numbered list with the found ones preselected. Picks live in `~/.jstack/config.json`, so reruns and `jstack upgrade` don't ask again. Backups of overwritten skills and replaced instructions files go to `~/.jstack/backup/<stamp>/<harness>/`, one place, instead of `.jstack-backup` folders inside skills folders and `.bak` files next to instructions files. The git hooks step went with the script; `make install-hooks` in the checkout does that job.

The table has five rows. Claude Code and Codex were verified on a real install. OpenCode's global skills folder is `~/.config/opencode/skills/` per its docs, plural, which corrected the second-hand `skill`. Pi's `~/.pi/agent/skills/` and `~/.pi/agent/AGENTS.md` match its docs. Cursor's `~/.cursor/skills` matches its docs, but `~/.cursor/rules/jstack.mdc` carries over from setup.py and Cursor's docs only describe project-level `.cursor/rules`, so that row is unverified. OpenCode, Cursor, and Pi also read `~/.claude/skills` or `~/.agents/skills`. Checked on OpenCode 1.18 with a fake home holding the same skill in both folders: it lists the skill once, keyed by name. Nothing to handle until a harness shows one twice.
