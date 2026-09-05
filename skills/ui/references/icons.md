# Icons

Rules for SVG icons, Heroicons, inline checkmarks, icon buttons, icon sizing, and lining icons up with text.

## Design rules

- Never generate raw SVG icons. Import from the project's icon library, or Heroicons if there isn't one yet
- Never wrap icons in decorative containers (colored squares, circles with backgrounds). Use the icon on its own
- Never scale icons. `viewBox="0 0 24 24"` is always `size-6`, `viewBox="0 0 20 20"` is `size-5`, `viewBox="0 0 16 16"` is `size-4`. If the icon looks too small, switch icon sets. Don't bump the size class
- Icons inline with `text-sm` text (checklists, feature items, comparison tables, inline labels) are always 16px/micro (`size-4`). 20px/mini icons (`size-5`) are only for navigation list icons
- Next to a text group (label plus supporting text), line the icon up with the first line using `items-start` or `items-baseline`. Never `items-center` on the group
- Application UIs (dashboards, settings, admin, sidebar nav, forms) only use Heroicons Micro (16px, `size-4`). Never 20px/mini or 24px/outline icons there

## Coding rules

- Use `size-{n} h-lh` on SVG icons to vertically center them with the text next to them. Set the `font-size` on a wrapper element, not top margins or manual alignment
- Use `fill-{color}` for filled icons and `stroke-{color}` for stroked icons. Never `text-{color}` with `currentColor` (a legacy v2 hack)
- Always add `shrink-0` to icons inside flex containers
