# Epistemics

How to talk about confidence when the evidence is old, partial, and sometimes contradictory. Code doesn't carry its own reasons. Those live in commits, PRs, tickets, docs, and chat, all incomplete and some missing. Pretend otherwise and the reader acts on a confident guess.

## Confidence tiers

Every claim in the final output sits in one tier. The tier decides its section and its phrasing.

### 1. Direct

An author wrote the reason down. Not "the code does X so the author wanted X". An actual sentence saying why.

- A PR description: "fixes the bug where users with more than 1000 items couldn't paginate".
- A ticket: "adding this because Acme asked for it in their security review".
- A code comment: "clamp to 100 because the upstream API rejects larger values".
- A design doc: "we chose A over B because we need persistence across restarts".
- A chat message from the author: "switching approaches, the old one was flaky in tests".

Phrasing: plain, present tense. "This exists because X." Cite it.

### 2. Supported

Several indirect pieces point the same way. Nothing says it outright, but the pattern makes it likely.

- PR title says "improve performance", the ticket is labeled perf, the surrounding commits all touch the same hot path.
- Tests added with the change all exercise very large inputs.
- The author's other PRs that week all mention the same incident.

Phrasing: confident but visibly derived. "The evidence points strongly to X." List the pieces.

### 3. Inferred

A reasonable reading of the context, with nothing supporting it directly. The reader should know it's your interpretation.

- The PR doesn't say why, but the incident channel puts the error in production and the fix merged the same day, so likely a hotfix.
- Retry count is 3, matching the convention elsewhere in the codebase.

Phrasing: hedged. "Appears to", "likely", "suggests", "is consistent with". Show the chain. "Given A and B, C seems likely because D."

### 4. Speculative

Plausible but thin, and other explanations fit as well.

- "This might be a workaround for a browser bug since fixed, but nothing from the time says so."
- "The threshold could match an SLA commitment, but no SLA doc mentions it."

Phrasing: explicitly a guess. "One possibility is X, with no direct evidence." Usually lives under competing hypotheses.

### 5. Unknown

You looked and didn't find out. A real and useful outcome.

Phrasing: say what you searched. "We searched the tracker for A and B, read the six PRs touching this file since 2023, and grepped for the threshold value. None gave a reason." That beats "we couldn't find out".

## Words that carry confidence

These claim Direct or Supported. Don't use them for inferences. The citation goes right next to them.

- because
- the reason is
- was designed to
- fixes, addresses, solves
- the team decided

## Words that hedge

For inferences. Use them freely in the inferred section.

- appears to, seems to
- likely, plausibly
- suggests, is consistent with
- one reading is
- may have been
- the evidence points toward

## Words to avoid

- obviously, clearly, of course. If it were, they wouldn't be asking.
- just, as in "it's just for performance". Dismissive, and usually hiding uncertainty.
- I think, I believe. You're weighing evidence, not giving an opinion. "The evidence suggests."

## Don't rationalize

Code that makes sense today may have been written for reasons that no longer apply, or for no good reason. Don't fit a clean story onto messy history.

- Don't assume the author did the right thing and work backward.
- Don't assume a consistent pattern was intentional. It might be copy-paste.
- Don't turn absence of evidence into evidence of absence. "Nobody mentioned security" doesn't mean it wasn't a concern.

## The user's guess

"Why do we do this, I assume for performance?" Don't confirm it. Treat it as one candidate and check the evidence. If it holds, say so with citations. If not, say so and say what the evidence does support.

## When sources disagree

Show both with citations. The ticket says "for customer X's compliance need". The PR says "cleaning up tech debt". Both might be true: the ticket motivated the work and the PR is the author's framing. Or one is wrong. Let the reader decide.

## When there's nothing

An honest "we don't know" tells the reader the answer isn't in the obvious places, that they'll need to ask a person, or that it isn't worth chasing. Name the gap: what question, which sources, what terms, what came back.

## Before finalizing

For every claim in found and inferred:

1. Does it have a citation? If not, move it down a tier.
2. Is the phrasing right for its tier? Direct can say because. Inferred can't.
3. Is the code being cited as evidence for its own intent? Remove or reclassify.
4. Is there a gaps section with specific gaps? If not, something's being hidden.
