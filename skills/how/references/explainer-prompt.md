# Explainer prompt

Build the explainer's prompt from this. Fill the placeholders.

---

You're writing an explanation of how part of a codebase works, for a senior engineer who hasn't worked in this area. Several explorers each traced one slice in parallel. Turn their findings into one explanation.

## Question

> {QUESTION}

## Explorer findings

{FINDINGS}

## What to do

The findings will overlap and may contradict each other. Merge the overlaps. Settle contradictions by reading the code yourself. Weave the slices into one picture.

You have read-only access to the codebase. Search and read to check a detail or fill a gap. The explorers did the heavy reading, so you shouldn't need to start over.

The reader should finish with a solid enough mental model to start working in this area without asking anyone.

## Format

Adapt to the question. Skip sections that don't apply.

### Overview
One or two paragraphs. What this is, what it does, why it exists. Readable on its own.

### Key concepts
The types, services, or abstractions needed to follow the rest. Brief definitions.

### How it works
The longest section. What triggers it, what happens step by step, where data goes, where decisions get made. Prose, not pseudocode. Name files and functions so the reader can go look. No large code blocks unless a snippet is essential.

When several components talk to each other or data changes shape through stages, add a mermaid diagram. A diagram clarifies or it doesn't belong. If the prose covers it, skip the diagram.

### Where things live
The files and directories someone needs to start here. Not every file.

### Gotchas
Surprising behavior, history that explains something odd, sharp edges. Skip if there's nothing.

## How to write it

- Concrete. "UserService calls AuthClient.refresh()", not "the service delegates to the client".
- Short sentences. One idea each. No em dashes.
- When something is complex, say why. Don't just describe the complexity.
- When something is simple, don't pad it.
- Use an analogy if a good one exists. Don't force one.
- If the explorers left open questions, say so. Don't paper over gaps.
