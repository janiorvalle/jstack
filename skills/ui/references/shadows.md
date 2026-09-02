# Shadows

These rules apply to shadows on cards, modals, popovers, dropdowns, buttons, and other raised surfaces, and to pairing shadows with borders.

- Don't pair a shadow with a solid gray border. Use `ring-1 ring-black/5` or `ring-1 ring-black/10` (or `950` of your neutral).
- Don't make a raised element (a card, modal, or popover with `shadow-*`) darker than the canvas under it. Use `white` or your lightest neutral, not `gray-100`/`gray-50`. Darker fills are fine on inset panels and wells that have no outer shadow.
