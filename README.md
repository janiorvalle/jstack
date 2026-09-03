# jstack

<p align="center">
  <img src="assets/hero.png" alt="jstack. how I work, stacked." width="840">
</p>

How I work with coding agents, written down as skills.

One flow from claiming a task to turning it in with proof. Twenty-four principles, each a single rule with a test. Sixteen workflows for the parts that need steps. A mode skill that ties it together. Every file is written to be read by an agent in Claude Code, Codex, Cursor, or anything else that loads skills from a folder.

The opinions are the point. A human stays in the merge seat. Every claim ships with evidence. The obvious solution wins over the clever one. If you don't agree with those, this isn't your stack, and that's fine.

## Install

```sh
git clone https://github.com/janiorvalle/jstack
cd jstack
python3 skills/setup-jstack/scripts/setup.py --agent auto --apply
```

`--agent auto` picks the harness you're running in. `--agent both` installs into Codex and Claude Code. Without `--apply` it's a dry run. It never deletes a skill it doesn't own, and it backs up anything it overwrites. It also fetches the vendored third-party skills from `vendor.json`, checks every tool in `tools.md`, and installs each tool's own skill. Add `--install-tools` to have it install missing tools too.

Restart your harness afterward so the skills load.

## What's in it

- `AGENTS.md` is the letter. Who's who, what we're doing, and the two or three rules that matter before you've read anything else. Setup installs it into your harness's user-level instructions file so the mode is always on.
- `skills/jstack-mode/` is the front door. The flow, the rules, and an index of every skill, generated from their description lines.
- `skills/*/` with `kind: principle` in the frontmatter are the principles. One rule each.
- The rest are workflows. `how`, `why`, `architect`, `arena`, `swarm`, `land-pr`, `worktree`, and so on.
- `tools.md` names the tools the flow expects to find installed and how to get them.
- `vendor.json` pins the third-party skills the stack depends on, fetched at setup time, never committed here.
- `decisions.md` is the record of choices made while building this, so nobody relitigates them.

## Development

```sh
make install-hooks   # gitleaks plus the verify script on every commit
make verify          # what CI runs
make index           # rebuild the mode's skill index after changing a description
```

See `CONTRIBUTING.md` for the shape of a skill and the CLA.

## Credit

The idea of a personal agent stack with a mode as the front door, small principle files, and a fan-out pattern for understanding code was inspired by [pstack](https://github.com/cursor/plugins/tree/main/pstack). Everything here is written from scratch.

## License

MIT.
