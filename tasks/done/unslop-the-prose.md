---
status: done
owner: agent:jstack-f1-lead
pr: https://github.com/janiorvalle/jstack/pull/40
files: [every *.md outside skills/agent-browser, skills/typescript-best-practices, and tasks/]
---
Problem: The skills and docs grew under review pressure and read like machine text: long sentences, hedges, repeated clauses, stories about how a rule came to be. Rules got harder to find.
Fix: Rewrite every markdown file except the two vendored skills with the voice skill. Shorter and plainer, every rule, command, path, name, and number kept.
Done when:
- make verify passes and the letter's workflow table is regenerated.
- A reviewer diffing each file finds no rule, command, path, or number lost.
- decisions.md drops a quarter or more; the setup, tracker, mode, and why skills and the letter's prose each drop a tenth or more.
Out of scope: the vendored skills, code, and anything under tasks/.
