---
status: doing
owner: agent:jstack-f1-lead
files: [every *.md outside skills/agent-browser, skills/typescript-best-practices, and tasks/]
---
Problem: The skills and docs grew under review pressure and read like machine text: long sentences, hedges, repeated clauses, stories about how a rule came to be. The rules are harder to find than they were.
Fix: Rewrite every markdown file except the two vendored skills with the voice skill. Shorter and plainer, with every rule, command, path, name, and number kept. Nothing an agent relied on disappears.
Done when:
- make verify passes and the letter's workflow table is regenerated.
- A reviewer diffing each file finds no rule, command, path, or number lost.
- Total words drop by at least a quarter.
Out of scope: the vendored skills, code, and anything under tasks/.
