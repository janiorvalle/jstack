# Border radius

Use these rules for rounded cards, panels, buttons, images, screenshots, nested surfaces, and anywhere radius consistency matters.

- Closely nested rounded elements need concentric radii. Spell out the relationship with CSS variables and `calc()` so the math holds, e.g. `rounded-(--radius) p-(--padding)` on the outer element and `rounded-[calc(var(--radius)-var(--padding))]` on the inner
- Image and screenshot radii use `min()` with viewport units instead of fixed `rounded-*` values, e.g. `rounded-[min(1vw,12px)]`. The radius should hit the intended value at full desktop width and shrink in proportion as the screen gets smaller
