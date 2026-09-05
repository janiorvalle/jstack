# Buttons

Rules for primary buttons, secondary buttons, CTAs, icon buttons, destructive actions, form actions, and touch targets.

## Design rules

- Button shadows follow shadows.md. Never pair `shadow-*` with a solid gray border. Use `ring-1 ring-black/5` or `ring-1 ring-black/10`
- A primary button with a ring never uses a reduced-opacity ring. Use a solid color matching the button background (e.g. `ring-indigo-600` on a `bg-indigo-600` button, not `ring-black/10`)
- Dangerous actions like "Delete" get a secondary or muted style by default. Use a primary style only when the dangerous action is the main action on the page or dialog (e.g. a confirm-delete dialog)
- One primary button per page. Scan the whole page and check that only one button uses a filled, solid primary style. Every other button is secondary: soft/muted (solid with opacity), outline, or ghost (text-only). Dialogs and modals count as their own page
- Never make a secondary button higher contrast than the primary. The primary is always the most prominent button
- Any button that isn't the page's primary submit or save action is an inline form action: change avatar, change photo, upload file, generate password, verify email, add item, resend code, and so on. These always get the smaller of the two button sizes and a secondary style. They are never the same height as the form's primary submit button

### Sizing

- Use less horizontal padding. `px-3 py-2` not `px-4 py-2`, `px-4 py-3` not `px-5 py-3`
- Application UIs (dashboards, settings, admin) use `text-sm` with compact padding, never `text-base`. Total rendered button height, including any outer wrapper or ring, stays within 28 to 38px. Count the `p-px` border wrapper, which adds 2px total
- At most 2 button sizes per application UI. Pick two heights at least 6px apart and use only those
- A button with a leading or trailing icon never gets symmetric `px-*`. Use `pl-*`/`pr-*`, with the icon side's padding equal to the vertical padding: `py-2 pr-3 pl-2` (left icon), `py-2 pr-2 pl-3` (right icon), `py-1.5 pr-2.5 pl-1.5` (left icon, compact)

### Focus styles

- Solid buttons need a custom focus ring. Use `focus-visible:outline-*` with `focus-visible:outline-offset-2`. Default to `focus-visible:outline-blue-500` if the project has no focus color yet

## Coding rules

- Small and icon buttons must meet the 48×48px minimum touch target. Make the button `relative` and add `<span class="absolute top-1/2 left-1/2 size-[max(100%,3rem)] -translate-1/2 pointer-fine:hidden" aria-hidden="true" />` as a direct child
