# Pricing cards

Rules for pricing tiers, pricing cards, pricing tables, plan comparisons, and any emphasized or "Popular" plan.

## Design rules

- Emphasize a card with its button styling and, if you want, a "Popular" or "Recommended" label. Don't give the whole card a different background color.
- Checkmarks in a feature list follow icons.md. Use `size-4 h-lh` so they center vertically with the text.

## Coding rules

- Don't pull the emphasized card out of the grid. It's a grid sibling, not its own section.
- Line up the buttons across all cards. Put `flex flex-col justify-between` on each card and wrap everything above the button in a single `<div>` so the button drops to the bottom.

```html
<div class="flex flex-col justify-between …">
  <div>
    <!-- name, price, description, features -->
  </div>
  <div>
    <button>Get started</button>
  </div>
</div>
```

- If the emphasized card is taller than its siblings, use CSS grid with explicit rows. No negative margins, no relative positioning. The gap rows decide how far the card pokes out. Normal cards sit in the middle row and the emphasized card spans every row.

```html
<!-- Pokes out top and bottom -->
<div class="{breakpoint}:grid-cols-3 {breakpoint}:grid-rows-[--spacing(6)_1fr_--spacing(6)] grid">
  <div class="{breakpoint}:row-start-2"><!-- normal card --></div>
  <div class="{breakpoint}:row-span-full"><!-- emphasized card --></div>
  <div class="{breakpoint}:row-start-2"><!-- normal card --></div>
</div>

<!-- Pokes out top only -->
<div class="{breakpoint}:grid-cols-3 {breakpoint}:grid-rows-[--spacing(6)_1fr] grid">
  <div class="{breakpoint}:row-start-2"><!-- normal card --></div>
  <div class="{breakpoint}:row-span-full"><!-- emphasized card --></div>
  <div class="{breakpoint}:row-start-2"><!-- normal card --></div>
</div>
```
