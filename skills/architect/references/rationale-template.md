# Rationale template

The prose beside the type sketch. One page. Replace the notes with content.

## Problem

One paragraph. What we're doing and what about the existing system makes the shape non-obvious. Name the constraints grounding surfaced: types to interop with, callers we can't break, invariants that cross our boundary.

## Usage, the caller's view

Write this first. The readme or quickstart the consumer reads, plus two or three realistic call sites. What they import, call, get back. The type sketch is derived from this. When they diverge, fix the sketch.

## Shape

The recommended design. Data structures first, then how data flows through the signatures. Name the decisions that carry weight. Which invariants are in types, where validation lives, what the system deliberately doesn't do. Say what the public interface hides, what stays exposed, and why the interface is no bigger than needed. Name the principle behind each decision. Don't restate it.

## Synthesis decision

Filled in by arena. Which candidate became the base and why, what was taken from each of the others, what was rejected and why.

## Tradeoffs accepted

One bullet each. "We accept X in exchange for Y." Include anything a future reader might mistake for an oversight.

## Alternatives considered

Required. At least one concrete alternative shape with one line on why it lost. Judge each on how much it hides behind how small an interface, not on implementation ease. Two or three when the design space had real contenders. One when the constraints forced the answer, phrased as "this was the only workable shape because". Different shapes, not flavors of one. Design alternatives, not the other runners' candidates.

## Open questions and risks

Things the human needs to weigh in on, and risks worth flagging before building. Phrase as questions so the answer is the resolution.

## Next step

The first thing to build against the sketch. One sentence.
