# Landing pages

Rules for landing pages and marketing pages: stacked sections, heroes, CTAs, pricing sections, feature sections, and keeping the whole page consistent.

- Reuse the same primary and secondary button styling across the whole page. If the hero uses a link-style secondary, every other section with a secondary action (CTA, pricing, etc.) uses a link-style secondary too
- Reuse the same font treatment (size, weight, color) when the same or a similar idea shows up more than once on a page. Match the existing instance exactly
- Use one container style across the whole page. Once it's set (outline, tinted, etc.), every later container matches
- Use one border radius for all containers at the same level. Panels, cards, and other sibling containers share it
- Use one column `gap-*` value across all multi-column sections. Card grids, split layouts, and any other multi-column section share the same gap. Check existing sections before adding a new one and match the value in use
- Never put a centered or center-constrained layout right below a left-aligned one. Use left-aligned instead, unless the section above ends with full-width containers that make a natural divide, a background color change separates them, or there's a visible divider between them
- Never have more centered heading groups than left-aligned ones on a landing page. Centered headings work best for hero sections, CTAs, and sections with symmetrical content under them (e.g. centered pricing cards, logo clouds). Default to left-aligned for feature grids, split layouts, and content-heavy sections
