---
status: open
---
Problem: Setup's bare line prompts confuse. Arrow keys spill escape codes, saved picks hide a harness added later, and a Homebrew-owned tool gets an update line that can't win on PATH.
Fix: With a terminal, setup becomes a guided flow on charm's huh: one screen per question, saved answers preselected, a plan and confirm at the end. Update offers use the updater that owns the binary. Flag path unchanged.
Done when:
- A rerun is one Enter when nothing changed; a new harness is offered.
- Arrow keys, space, Enter, Esc work on every screen.
- A Homebrew TruffleHog is offered brew upgrade, skippable.
- Flag path tests pass unchanged.
Out of scope: the agent path, the letter, the skills.
