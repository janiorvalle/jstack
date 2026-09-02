# Navigation

Rules for sidebar nav, header nav, mobile menus, tabs, tab bars, vertical menus, active states, and current-page indicators.

- Every app needs a mobile navigation menu on small screens, whether the desktop nav lives in a header or a sidebar. Use a dialog or disclosure panel with a hamburger toggle. Hide the desktop nav with `hidden lg:flex` (header) or `hidden lg:block` (sidebar) and show the mobile menu below `lg:`
- Never use a high-contrast or primary-color background for active nav items. Use darker text color, a soft/muted background, or both
- Never change `font-weight` between nav item states (default, hover, active). Use color and background changes only
- Horizontal menus (tabs, tab bars, pill navs) must never overflow the parent container. Use horizontal scrolling when items don't fit
- Never use icons in top header horizontal navigation links. Use text-only links
- When centering nav links on the page (not just between the side items), use a three-section flex layout: `<div class="flex flex-1 items-center">` for the left section (logo), the nav links at their natural width (no `flex-1`), and `<div class="flex flex-1 items-center justify-end">` for the right section (actions). The matching `flex-1` gutters push the centered group to the true page center. Do the same when centering a logo. Keep the logo at its natural width and use `flex-1` on the side sections so it centers on the page rather than between the items around it
