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
