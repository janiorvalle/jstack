# Contributing

Thanks for pitching in. jstack is opinionated on purpose, so most contributions are one of three things: a skill that's wrong about how something works, a skill that reads like a machine wrote it, or a gap the flow doesn't cover.

## Setup

You need git, Go 1.24 or newer, python3, and gitleaks. Nothing else.

```sh
make install-hooks
```

That points git at `.githooks/`, where the pre-commit hook runs gitleaks and then the same verify script CI runs.

## The gate

```sh
make verify
```

It formats, builds, vets, and tests the binary, then checks that every skill has frontmatter with a name matching its folder, that the workflows table in `AGENTS.md` matches the description lines, and that nothing carries an em dash or names a specific harness. If you change a description line, run `make index` and commit the result.

## Writing a skill

Read `skills/voice/SKILL.md` first. Every file here follows it. Then read two or three existing skills to get the shape.

- One skill, one job. A principle is one rule with a test. A workflow is one procedure with steps.
- Harness-agnostic. Say "spawn read-only subagents" and "search the code", never a tool name from one harness.
- Tracker-agnostic. Say "the tracker", never a specific one.
- Plain words. Short sentences. No em dashes. Contractions.
- Frontmatter is `name`, `description`, and for principles `kind: principle`. The description line is what the mode's index shows, so make it say when the skill applies.

## Pull requests

One task, one branch. Squash merges only. Commit titles in conventional commit style naming the outcome. The PR description opens with the problem, then the fix, and ends with a line naming the model and harness that made the change if one did.

## Licensing

Contributions are licensed under the MIT License and the contributor license agreement below. The CLA check runs on a contributor's first pull request.

## Contributor License Agreement

By commenting `I have read the CLA Document and I hereby sign the CLA` on a pull request, you grant Janior Valle and recipients of this project a perpetual, worldwide, non-exclusive, royalty-free, irrevocable license to use, reproduce, modify, display, perform, sublicense, and distribute your contribution and derivative works under the project's license.

You represent that you are legally entitled to grant this license and that, to your knowledge, the contribution is your original work or is submitted with permission. You are not expected to provide support for the contribution unless you agree to do so separately.
