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

`vendor.json` stays the pin record. The committed copy is verbatim, license file alongside, and nobody edits it, because the weekly vendor-bump workflow overwrites the folder. A skill's version is the last upstream commit that touched its folder, not the repo head, so an unrelated upstream commit doesn't open a PR. Vendored text is upstream's voice, not ours, so `verify.py` only checks that a SKILL.md exists and `build-index.py` leaves them out of the letter's table. `tools.md` still names agent-browser, since setup has to install the tool.

Bump PRs are opened with the workflow's own token, and GitHub doesn't start workflows for those, so CI doesn't run on them. The reviewer runs `make verify` before merging. A skill folder can't fail verify anyway, short of losing its SKILL.md.
