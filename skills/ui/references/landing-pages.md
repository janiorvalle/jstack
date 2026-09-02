# Landing pages

Rules for landing pages and marketing pages: stacked sections, heroes, CTAs, pricing sections, feature sections, and keeping the whole page consistent.

- Reuse the same primary/secondary button styling across the whole page. If the hero uses a link-style secondary, every other section with a secondary action (CTA, pricing, etc.) must use a link-style secondary too
- Reuse the same font treatment (size, weight, color) when the same or a similar idea shows up more than once on a page. Match the existing instance exactly
- Use the same container style across the whole page. Once a container style is set (outline, tinted, etc.), every later container should match
- Use the same border radius for all containers at the same level. Panels, cards, and other sibling containers on a page should share one radius
- Use the same column `gap-*` value across all multi-column page sections. Card grids, split layouts, and any other two-column or multi-column section must share the same gap. Check existing sections before adding a new one and match the value already in use
- Never put a centered/center-constrained layout right below a left-aligned layout. Use left-aligned instead, unless: the section above ends with full-width containers that make a natural divide, a background color change separates them, or there's a visible divider between them
- Never have more centered heading groups than left-aligned ones on a landing page. Centered headings work best for hero sections, CTAs, and sections with symmetrical content under them (e.g. centered pricing cards, logo clouds). Default to left-aligned for feature grids, split layouts, and content-heavy sections
