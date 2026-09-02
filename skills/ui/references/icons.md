# Icons

Rules for SVG icons, Heroicons, inline checkmarks, icon buttons, icon sizing, and lining icons up with text.

## Design rules

- Never generate raw SVG icons. Import from the project's existing icon library, or use Heroicons if there isn't one yet
- Never wrap icons in decorative containers (colored squares, circles with backgrounds). Use the icon on its own
- Never scale icons. `viewBox="0 0 24 24"` always uses `size-6`, `viewBox="0 0 20 20"` uses `size-5`, `viewBox="0 0 16 16"` uses `size-4`. If the icon looks too small, switch to a different icon set. Don't bump the size class
- Always use 16px/micro icons (`size-4`) when inline with `text-sm` text: checklists, feature items, comparison tables, inline labels. Only use 20px/mini icons (`size-5`) for navigation list icons
- For icons next to a text group (label + supporting text), line the icon up with the first line/label using `items-start` or `items-baseline`. Never `items-center` on the group
- In application UIs (dashboards, settings, admin, sidebar nav, forms), only use Heroicons Micro (16px, `size-4`). Never use 20px/mini or 24px/outline icons in application UIs

## Coding rules

- Use `size-{n} h-lh` on SVG icons to vertically center them with the text next to them. Set the `font-size` on a wrapper element instead of using top margins or manual alignment
- Use `fill-{color}` for filled icons and `stroke-{color}` for stroked icons. Never use `text-{color}` with `currentColor` (that's a legacy v2 hack)
- Always add `shrink-0` to icons inside flex containers
