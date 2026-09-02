---
name: fake-the-affordance-not-the-contract
description: "Use when building anything an agent will use: an MCP, a CLI, an API, a sandbox, a mock environment. It's fine to give the agent a familiar interface that's simpler underneath than it looks. It's never fine to fake durability, isolation, security, persistence, or production readiness."
disable-model-invocation: true
kind: principle
---

# Fake the affordance, not the contract

Agents need tools and instructions. They don't need accurate implementation details of those tools. They don't need real environments. Agents like working a certain way, and if we give them that way, they work well. What's underneath can be a mirage as long as the agent gets what it expects and we get outputs that work.

This sounds wrong until you've built for agents. Then it's obvious.

## What you can fake

- **A familiar interface over a different implementation.** An agent expects a filesystem, give it something that answers like a filesystem. It expects a database, give it something that takes SQL. What's behind it is your business.
- **A simpler world than the real one.** A "deploy" command that writes to a staging directory. A "send" that logs. A "user" that's a fixture. The agent gets to work the way it knows how.
- **Affordances it asks for.** If agents keep reaching for a command that doesn't exist, build the command. It doesn't have to do what the real one would. It has to give the agent what it needed to move on.

## What you can never fake

The contract. These are promises the agent will act on, and if the promise is false the agent's work is wrong in ways nobody will catch until it matters.

- **Durability.** If something says it saved, it's saved. If it isn't, say it isn't.
- **Isolation.** If something says this is your sandbox, nothing else can see it or touch it.
- **Security.** If something says this is safe, it's safe. A fake auth check is worse than none.
- **Persistence.** If the agent will assume it's still there next session, it's still there.
- **Production readiness.** If it looks shippable, it's shippable. A mock that could be mistaken for the real thing needs to say it's a mock.

## The test

Ask what the agent will do next based on what it just saw. If a false belief about the implementation changes nothing about what it does next, fake it freely. If a false belief would make the agent skip a step, trust something it shouldn't, or tell a human something untrue, that's contract, and it has to be real.
