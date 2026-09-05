# Prose content

Rules for raw HTML you can't put classes on: markdown output, CMS content, database content, blog posts, articles, and docs.

- Don't use the `@tailwindcss/typography` plugin. Write a `.prose` class that styles the raw elements (headings, links, lists, code blocks, images, and so on) in plain CSS using Tailwind's theme variables (`var(--color-*)`, `var(--text-*)`, `var(--font-weight-*)`, `var(--radius-*)`, `--spacing(*)`, `--alpha()`). Use `@variant dark { … }` and `@variant hover { … }` for dark mode and hover. Use `* + *` for the vertical gap between elements. Style every element that could show up: `h1` to `h6`, `p`, `a`, `ul`, `ol`, `li`, `pre`, `code`, `img`, `strong`, `blockquote`.
- Put the `.prose` class on the element that wraps the rendered HTML: `<div class="prose">` around a blog post, markdown output, CMS markup, or any HTML where you can't add classes to the elements inside.
- Default prose body text to `var(--text-base)` (`16px`). Only go to `var(--text-lg)` (`18px`) or bigger if someone asked for it or the project already uses that size for body text elsewhere.
- Don't set `max-width` inside the `.prose` CSS. Constrain width with a `max-w-[*ch]` class next to `prose` in the markup (e.g. `<div class="prose max-w-[65ch]">`). Use `60ch` to `90ch`, matched to the content widths the site already uses.
- Set prose body `line-height` to at least `1.75` times the font size, e.g. `--spacing(7)` for `var(--text-base)`.
- Use `text-pretty` on blog post and article titles, not `text-balance`.
- If the article title (`h1`) is a sans-serif, use that same sans-serif for every subheading (`h2` to `h6`). Don't mix a sans-serif title with serif subheadings.
