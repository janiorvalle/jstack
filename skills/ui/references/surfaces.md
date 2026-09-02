# Surfaces

These rules apply to cards, wells, borders, dividers, white space, recessed backgrounds, and any other way of grouping content.

- Don't default to white cards on a gray background. Put content straight on a white background, or use white cards with just a `border` on a white background.
- Pick the surface treatment from the information hierarchy. White space alone for tightly related items. Subtle borders or dividers for sibling content that needs separating. Wells (recessed backgrounds like `bg-gray-50`) for secondary or nested content. Cards with borders or shadows for standalone, interactive, or clearly different items.
- Use the lightest separation that still works. Whitespace first, then subtle borders or dividers, then cards. Don't jump straight to cards.
- Save cards for content that's interactive on its own (clickable to navigate) or holds a fundamentally different kind of content.
- Use subtle top borders or vertical dividers for sibling items in a shared context: stat grids, metric rows, dashboard KPIs.
- For divider-separated items, middle items get equal padding on both sides of the divider (`px-*`). The first item in a row gets only `pr-*` (no `pl-*`) and the last gets only `pl-*` (no `pr-*`). For horizontal dividers, the first item gets only `pb-*` (no `pt-*`) and the last gets only `pt-*` (no `pb-*`). When the column count changes at a breakpoint, reset the padding for the new first and last items. In a 4-column grid that becomes 2-column, items 1 and 3 are now row-starts (no `pl-*`) and items 2 and 4 are now row-ends (no `pr-*`). Use responsive prefixes like `sm:pl-0` or `lg:pr-0` to override at each breakpoint.
- Dividers have to be redone at each breakpoint where the column count changes. Use `nth-child` to target items that aren't in the first column. 2 columns use `[&:nth-child(2n)]:border-l-*`. 4 columns use `[&:not(:nth-child(4n+1))]:border-l-*`. Change the pattern at each breakpoint to match the column count. When it collapses to one column, drop the vertical dividers and add horizontal dividers between rows (`border-t-*` on every item except the first).
- Whitespace alone is enough when the content already contrasts on its own (big numbers next to small labels, bold headings next to body text).
- Don't use solid colors for dividers. Use opacity-based colors like `divide-gray-950/5` or `border-gray-950/10` instead of `divide-gray-200` or `border-gray-300`.
