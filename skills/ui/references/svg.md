# SVG

These rules apply to inline SVG, SVG color styling, `fill`, `stroke`, `currentColor`, and SVG markup in general.

- Leave `xmlns` off inline `<svg>` elements in HTML and JSX. You only need it in a standalone `.svg` file.
- Color SVGs with Tailwind classes (`fill-*`, `stroke-*`, or `text-*` with `fill="currentColor"`/`stroke="currentColor"`), not hardcoded color attributes or inline ternaries. Switch colors with `data-*`/`aria-*` variants or conditional classes.
- Don't put `fill="currentColor"`/`stroke="currentColor"` attributes and `fill-*`/`stroke-*` classes on the same element. The attribute fights the class. Use `fill-current`/`stroke-current` to inherit the text color, or drop the attribute when you're using a specific color class like `fill-zinc-400`.
