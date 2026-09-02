# Dark mode

Use these rules for dark-mode styling, light-to-dark conversion, dark-mode contrast audits, dark-mode images, and dark-mode SVGs.

## Design rules

- Dark mode means keeping the same contrast ratios as light mode, not inverting colors
- Dark mode doesn't have to keep every detail of the light design. It just has to look good
- Default dark mode to the operating system's `prefers-color-scheme` setting (Tailwind's built-in `dark:` behavior). Only add a manual toggle when the user asks for one
- Remove all shadows in dark mode with `dark:shadow-none`
- On a dark-only site, add the `scheme-only-dark` class to `<html>` or the top-level element. That makes native things like scrollbars, form controls, and `color-scheme` render dark

## Component rules

- Never keep large branded or colored panels in dark mode. Use the same background color and add a light divider between sections instead
- Style cards only slightly lighter than the page background (e.g. `dark:bg-gray-900` on a `dark:bg-gray-950` page). Add `dark:inset-ring dark:inset-ring-white/5` for definition
- Make decorative quote marks in testimonials very faint (e.g. `dark:text-white/5`)
- Never mix heading text colors in dark mode (e.g. dark gray plus brand color). Use one light color like `white` or `gray-100` for every heading

## Raster image rules

- When you add or improve dark mode, audit the page for raster images that need dark versions: photos, screenshots, product mockups, decorative backgrounds, textures, and rasterized illustrations
- Never use CSS filters (`invert`, `brightness`, `contrast`, `opacity`) as the final dark-mode treatment for a raster image. Always create real dark-mode image files
- Generate dark-mode raster variants with the dark-mode skill, which uses your harness's image generation to create or edit the raster asset

## SVG rules

- For inline `<svg>` elements, style dark mode with Tailwind `dark:*` classes (e.g. `dark:fill-*`, `dark:stroke-*`, `dark:text-*`)
- For external SVG files loaded through `<img>`, always create a dark version next to the original (e.g. `logo.svg` and `logo-dark.svg`). Never swap in CSS filters (`invert`, `brightness`) or opacity tweaks for a real dark variant
