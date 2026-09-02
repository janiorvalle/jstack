# Rubric

Apply the lenses that fit. Not every lens fits every change.

## Correctness

Does it do what the intent says?

- Edge cases. Empty, null, boundaries, concurrent access.
- Errors. Caught, propagated, or silently swallowed?
- Off by one, coercion, overflow, encoding.
- State. Races, stale closures, dangling references.
- Happy path and sad path both work?
- Runs twice or crashes halfway. Does it converge, or does the outcome depend on what got left behind?
- Shared mutable state. Serialized by structure, or by a convention that won't hold?

When you find a possible bug, trace the path. Don't say "this could be null". Show the call chain that makes it null.

## Cause or symptom

Is this fixing the problem or covering it?

Read beyond the diff. Callers, callees, types, siblings. Understand why the code exists before judging whether the change hits the right layer.

- A guard that masks a broken invariant.
- Retry logic hiding a broken contract.
- A cast silencing a modeling error.
- A fix in module A that belongs in module B's contract.
- A comment or convention where a type, lint rule, or runtime check would make the wrong thing impossible.

## Fit

Does it belong in the system it's in?

- Validation at the edges, or scattered through logic?
- Mixing orchestration with low-level detail?
- New coupling that makes the next change harder?
- Data structures matching how the data is actually accessed?
- Bolted on, or does it read as if the design always accounted for it?
- A new API with the old one kept alive for no external consumer?

Don't penalize simple code for lacking abstraction. Duplication beats a premature abstraction.

## Verification

Can you tell it works from reading it?

- Tests of behavior, not implementation details?
- For a bug fix, a test for the bug?
- For an integration edge, the full path tested?
- Checking the real value, or a proxy like a timestamp or cached state?
- Delegated work verified by its artifacts, or by its summary?

## Complexity

Is the complexity earning its keep?

- Could be simpler without being wrong.
- An abstraction with one caller.
- Configuration for cases that don't exist.
- Dead code, unused imports, leftover parameters.
- "Just in case" paths with no callers.
- Compatibility scaffolding from a migration that's finished.
- Every feature, control, and option earning its place.

Simpler is better unless simpler is wrong.

## Structure, be ambitious

Don't stop at "this could be a bit cleaner". Look for the move that makes whole branches, helpers, modes, or layers disappear. If complexity can be deleted rather than rearranged, push for that.

- A file crossing from under a thousand lines to over is a strong smell. Ask whether it should be split first.
- New ad hoc conditionals or one-off branches dropped into an unrelated flow are a design problem, not a style nit. Push the logic into a helper, a state machine, or its own module.
- Thin wrappers, identity helpers, pass-throughs. Indirection with no clarity bought.
- Unnecessary optionality, `unknown`, `any`, or cast-heavy code where a clear type boundary could exist.
- Logic leaking out of its canonical layer, or a bespoke one-off where a shared helper already exists.
- Independent work serialized for no reason, or related updates that can leave state half-applied.

Prioritize structural regressions and missed simplifications, then tangled branching, then boundary and type concerns, then the small stuff. A few high-conviction findings beat a long list of cosmetic ones.

## Security

Only what you can trace. "Could be an injection vector" without the input path isn't useful.

- User input reaching SQL, shell, eval, or innerHTML without sanitizing.
- Auth gaps on new endpoints.
- Secrets in code, logs, or error messages.
- Check-then-use races on security paths.
