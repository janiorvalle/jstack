# Design red flags

Screen every candidate before picking. A red flag is a reason to revise or reject the shape.

## Shallow module

A big interface hiding little. Judge depth by how much capability sits behind the public interface relative to its size. Prefer a simple interface backed by substantial behavior.

Not the same as a deep call chain. A deep chain scatters understanding across layers. A deep module concentrates it behind one interface.

Signs. Callers coordinate several methods to do one thing. Public options expose internal stages. Learning the interface doesn't save the caller from learning the implementation.

## Leaked internals

More than one module depends on the same internal decision. A representation, policy, or protocol detail shows up in several places, so changing it means coordinated edits.

Re-exporting transport or wire types is leakage. Parse external data into domain types behind the interface. Keep storage schemas, framework objects, and protocol details private.

## Organized by execution order

Modules split by when things run, load then validate then transform then save, instead of by what they know. The same representation and its invariants get repeated at every stage boundary.

Group by knowledge and ownership. Methods that run at different times still belong together when they protect the same decisions.

## Pass-through

A method that forwards the same arguments to another method with the same shape. A layer that hides nothing.

Remove it, or move the responsibility to the module that can finish the job. Keep a forwarding boundary only when it adds policy, adaptation, or a genuinely different abstraction.
