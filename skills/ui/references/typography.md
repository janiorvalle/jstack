# Typography

These rules apply to text sizes, line heights, heading styles, font weights, tracking, text width, `text-pretty`, `text-balance`, and eyebrow text.

## Design rules

- Don't use `text-xs` for body text, paragraphs, or general page content. The smallest body text size is `text-sm`, and only at `sm:` or larger. The mobile default is at least `text-base` (16px).
- Don't use `font-bold` on headings. Use `font-semibold` or `font-medium`.
- Don't add `leading-*` or a line-height modifier to headings. Use Tailwind's default line-height (e.g. `text-6xl`, not `text-6xl/tight`).
- Use `text-balance` on headings and `text-pretty` on paragraphs.
- Add `tracking-tight` to headings bigger than `text-xl`, unless the font is a condensed headline font (its tracking is already tight).
- Don't use `uppercase` on eyebrow text unless it's in a monospace font. When you do use `uppercase` with a monospace font, always add `tracking-wide`.

## Coding rules

- Constrain text width with `max-w-[*ch]` right on the element. See heading-groups.md for the value per `text-*` size.
- Always use the official Inter variable font (`InterVariable`) with `font-display: swap`. Turn on OpenType features with `font-feature-settings` (e.g. `cv02`, `cv03`, `cv04`, `cv11`, `ss01`, `ss03`).
- Always read custom-fonts.md before using a custom font.
