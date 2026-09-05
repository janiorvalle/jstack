# Responsive design

Rules for mobile, tablet, and desktop layouts, breakpoints, container queries, and anything that overflows, wraps, clips, or gets cramped on a narrow screen.

## Design rules

- Every layout works from mobile to desktop. Use responsive breakpoint classes (`sm:`, `md:`, `lg:`, etc.) to change grid columns, spacing, font sizes, and visibility per screen size.
- Multi-column desktop layouts (sidebars, secondary navigation, filter panels) collapse to one column on small screens. Use a mobile menu, disclosure, or another compact pattern. Don't just squeeze the columns.
- Body text, subheadings, form controls, and icons are larger on mobile and shrink at `sm:`. Write the mobile (larger) size as the default and the desktop (smaller) size with `sm:` (e.g. `text-2xl/8 sm:text-xl/8`, `text-base/7 sm:text-sm/6`, `text-lg/6 sm:text-sm/6`, `size-5 sm:size-4`, `py-2.5 sm:py-1.5`). This covers body text, subheadings, stat values, form input labels, badges, buttons, select/input padding, and icons. It does not cover h1s. Page titles stay the same size or get smaller on mobile, never bigger.
- Body text is at least `text-base` (16px) on mobile. `text-sm` is only okay at `sm:` or larger (e.g. `text-base/7 sm:text-sm/6`, never `text-sm/6` on body copy without a breakpoint prefix).

## Coding rules

- Use container queries (`@container`) for component-level responsiveness: anything whose layout depends on the space it has rather than the viewport (e.g. dashboard widgets, feature cards, pricing tiers, testimonial grids).
- Put the `@container` element as close to the responsive content as you can, ideally the direct wrapper around the items. Never on a page-level container.
