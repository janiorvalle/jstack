# Border radius

Rules for rounded cards, panels, buttons, images, screenshots, nested surfaces, and anywhere radius consistency matters.

- Closely nested rounded elements need concentric radii. Write the relationship with CSS variables and `calc()` so the math holds: `rounded-(--radius) p-(--padding)` on the outer element, `rounded-[calc(var(--radius)-var(--padding))]` on the inner
- Image and screenshot radii use `min()` with viewport units, not fixed `rounded-*` values, e.g. `rounded-[min(1vw,12px)]`. The radius hits the intended value at full desktop width and shrinks in proportion on smaller screens
