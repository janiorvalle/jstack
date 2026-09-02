# Custom fonts

Use these rules when loading custom fonts, registering font theme variables, and applying display and body font utilities.

- Load a custom font before you use it. Add `<link>` tags in the HTML `<head>` (preferred). If there's no `<head>` to edit, put `@import url('…');` at the top of the CSS file instead
- Register fonts you use often in the CSS `@theme` block, e.g. `--font-display: "Oswald", sans-serif;`. You can also set `--font-display--font-feature-settings` and `--font-display--font-variation-settings` for fine-tuning
- Register headline and display fonts as `--font-display`, which creates a `font-display` utility. Use `--font-sans` for body and UI fonts and `--font-display` for fonts that only appear on headings and display text. Put `font-display` on headings and `font-sans` on the body
