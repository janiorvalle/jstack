# Section layout

Rules for page sections, constrained containers, centered versus left-aligned layouts, section padding, grids, and lining up stacked content.

## Design rules

- Left-aligned sections line up with the page container edge. Don't use a narrow `max-w-*` plus `mx-auto`. Use a page-level `max-w-*` and constrain the inner content on its own.
- Stacked sections with the same proportions line up their edges. A 1/2-width card grid and a 1/2-width split below it share the same column edges. Use the same grid definitions and gap values so the edges match as you scroll.
- Don't nest max-width constraints on grids or lists that fill their container. A feature grid or icon list that spans the full constrained width gets no narrower `max-w-*`. Its content lines up with the page container edges, not floating in the middle. A nested `max-w-*` is fine on self-contained units meant to feel bounded (pricing cards, forms, comparison tables, centered media).

## Coding rules

- Use a two-element pattern for constrained page sections. The outer element owns the background and vertical padding. The inner element owns max-width, centering, and horizontal padding:

  ```html
  <... class="{vertical-padding}">
    <... class="{max-width} mx-auto {horizontal-padding}">
      ...
    </...>
  </...>
  ```

  Use this on every section of a landing page so content edges line up as you scroll.
