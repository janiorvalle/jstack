# Runner prompt

Give this to every arena runner during the sketch phase, with the task, the grounding, its own working directory, and where to write. The directory is a worktree when possible. What matters is that candidates can't see each other.

---

You're producing one candidate design. Others are producing theirs at the same time, on different models. Output a design package: a type sketch, function signatures, a module map, and a rationale shaped by `rationale-template.md`.

The orchestrator compares candidates on these. Apply them.

- **Caller's usage first.** Write the readme-style usage and two or three real call sites before the types. Then derive the types from them. The usage is the spec. When they disagree, fix the types.
- **Data shapes first.** Get the core types right and the code becomes obvious. Trace each dominant access pattern through the structure. If the answer is "we'll add a map or a cache later", the structure is wrong.
- **Small surface, deep behind it.** Compare what the public interface hides against how big it is. Prefer a simple interface that pulls complexity into the callee, even if the callee gets less simple. No transport or wire types on the public surface. Parse into domain types behind it.
- **Shared state.** If two actors might both write, ask what happens. If the answer isn't "nothing", give each its own state and merge when reading.
- **Boundaries visible.** `not implemented` for bodies, pseudocode for tricky logic, doc comments stating intent and what must stay true. A reader should trace data from input to output from the types and signatures alone.
- **Invariants in types.** A type that can't be misused beats a runtime check beats a comment.
- **Validate at the edges, trust inside.** Business logic as pure functions. The wiring stays thin.
- **One source of truth per invariant.** Derive, don't sync.
- **Safe to rerun** where it applies. What happens if this runs twice, or dies halfway?
- **Short call chains.** If tracing the flow takes more than three files, flatten it.

Produce the best design your model can make. Don't hedge toward what the others might do. The differences between candidates are the signal. Converging on a safe middle defeats the point.
