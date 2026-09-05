# Lead judgment

The reviewers have reported. Don't aggregate. Filter, add context, decide.

## Why this step exists

Adversarial reviewers are useful because they're aggressive, and aggression without context is noise. They saw a diff and a paragraph. They don't know what was tried and rejected, what constraints live outside the code, which parts are temporary scaffolding, or what the next PR in the stack handles. You do. Use it.

## Filters

**Nitpick gravity.** Reviewers fill their review. With no critical findings, nits get inflated. If a reviewer's findings are all nits and preferences, the code is probably fine. Say so.

**Hypothetical versus actual.** "What if someone passes null" is a finding only if a caller can. Trace the call site. If the input is validated upstream or the types prevent it, dismiss. Reviewers working from a diff can't always see the chain. You can.

**Premature abstraction.** Reviewers love to extract functions and add interfaces. Does this code need to change in a second way? If not, the abstraction is premature. Inline code that works beats an abstraction that's overkill.

**"I would have done it differently."** The most common false positive. Not actionable unless the reviewer shows a concrete problem with the current approach. Dismiss, and say why.

**Missing context.** Suggesting changes to code the author didn't touch. Flagging patterns consistent with the rest of the codebase. Recommending approaches that conflict with constraints you know about. Honest mistakes from limited information. Dismiss gracefully.

## When they're right

Don't dismiss findings because they're uncomfortable. That's the whole point. A finding deserves attention when:

- More than one model flagged it independently.
- It names a concrete execution path, not a hypothetical.
- It reveals a gap in your own model of the code.
- You read it and think "yeah, actually".

Security and correctness findings get extra scrutiny even from a single model.

## Calibration

A good verdict is useful, not exhaustive. The human reads act-on, fixes those, and ships with confidence. More than five items in act-on means you aren't filtering.

The dismissed section is how the human checks you. Showing what you rejected and why lets them override you where they disagree. That's worth more than hiding it.
