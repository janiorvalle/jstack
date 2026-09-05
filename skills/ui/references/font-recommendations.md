# Font recommendations

Use these when someone asks for help picking a font or wants to try different fonts across design variations. Don't push them otherwise.

## General guidelines

- Default to Inter for body/UI text unless the user is specifically exploring other options
- Always recommend sans-serif fonts unless the user explicitly asks for serif or another non-sans-serif font, uses words like "sophisticated" or "editorial", or the project clearly calls for it (e.g. luxury brand, literary magazine, fashion editorial)

## By purpose

### Body and UI

Good for body copy, app interfaces, and general use. Most work for headings too.

**Sans-serif:**

- **[DM Sans](#dm-sans)**, low-contrast geometric, large x-height, very readable at small sizes
- **[Figtree](#figtree)**, friendly geometric with curved letterforms, warm and approachable
- **[General Sans](#general-sans)**, compact rationalist, space-efficient, good for dense UI
- **[Geist](#geist)**, Swiss-inspired, minimal and precise, built for UI
- **[Host Grotesk](#host-grotesk)**, uniwidth (weight changes don't shift layout), good for nav/tabs/buttons
- **[Inter](#inter)**, clean and highly legible, the default pick for screens
- **[Instrument Sans](#instrument-sans)**, geometric neo-grotesque, clean and technical
- **[Mona Sans](#mona-sans)**, GitHub's neo-grotesque, strong industrial feel
- **[Satoshi](#satoshi)**, modernist with personality, double-storey `a` and `g`

**Serif:**

- **[Lora](#lora)**, well-balanced contemporary serif, reads well at body sizes, subtle brush-stroke contrast

### Headlines and display

Best for headings and large display text. Many of the body/UI fonts above work for headings too. These are especially strong for display, or display-only.

**Sans-serif:**

- **[DM Sans](#dm-sans)**, low-contrast geometric that scales up cleanly for headlines and still reads well at body sizes
- **[Fixel Display](#fixel-display)**, geometric-humanist hybrid, display sizes only
- **[Geist](#geist)**, Swiss-inspired precision that looks sharp and authoritative at headline sizes
- **[Inter](#inter)**, clean and versatile, with a Display optical size variant that kicks in automatically at larger sizes
- **[Mona Sans (wide)](#mona-sans)**, Mona Sans with the `"wdth"` axis cranked up, strictly for headlines
- **[Satoshi](#satoshi)**, double-storey `a` and `g` give headlines personality without losing modernist discipline

**Serif:**

- **[Instrument Serif](#instrument-serif)**, high-contrast editorial serif, pairs with a sans-serif body for a premium feel

### Monospace

For code snippets, inline code, or a technical/developer look.

- **[Geist Mono](#geist-mono)**, Vercel's monospace, pairs naturally with Geist
- **[IBM Plex Mono](#ibm-plex-mono)**, IBM's monospace, versatile and highly legible

---

## Font details

### DM Sans

Low-contrast geometric with open apertures and a large x-height. Single-storey `a` and `g`, straight-legged `R`. Very readable at small sizes, so it's a good body font next to other headline fonts. Works for headlines too.

- **Source:** load from Google Fonts (`family=DM+Sans:opsz,wght@9..40,100..1000`)
- **Registration:** register in `@theme` as `--font-sans: "DM Sans", sans-serif;`
- **Pairs with:** Inter, Geist

### Figtree

Friendly geometric with distinctive curved `t`, `f`, and `y`. Warm without being playful. Monolinear stroke. Good for approachable designs. Works for headlines and body.

- **Source:** load from Google Fonts (`family=Figtree:wght@300..900`)
- **Registration:** register in `@theme` as `--font-sans: "Figtree", sans-serif;`
- **Pairs with:** Inter, Geist, DM Sans

### Fixel Display

Geometric-humanist hybrid with open letterforms and wide proportions. The Display variant is tuned for larger sizes. Headlines and display text only, never body copy.

- **Source:** must be self-hosted. Download from `https://fixel.macpaw.com`
- **Registration:** register in `@theme` as `--font-display: "Fixel Display", sans-serif;`
- **Pairs with:** Inter, Geist, DM Sans

### Geist

Swiss-inspired sans-serif by Vercel. Minimal, precise, built for UI. Works for body copy, app UI, and headings.

- **Source:** load from Google Fonts (`family=Geist:wght@100..900`)
- **Registration:** register in `@theme` as `--font-sans: "Geist", sans-serif;`
- **Pairs with:** Inter, DM Sans

### Geist Mono

Monospace by Vercel. Good for technical or developer-oriented sites, code snippets, and inline code.

- **Source:** load from Google Fonts
- **Registration:** register in `@theme` as `--font-mono: "Geist Mono", monospace;`

### IBM Plex Mono

IBM's monospace. Versatile and highly legible. Works for code snippets, technical content, and developer-oriented sites.

- **Source:** load from Google Fonts (`family=IBM+Plex+Mono:wght@400;500;600;700`)
- **Registration:** register in `@theme` as `--font-mono: "IBM Plex Mono", monospace;`

### General Sans

Compact rationalist sans-serif with small apertures and a disciplined, closed feel. Space-efficient, so it suits dense UI and tight layouts. Works for headlines and body.

- **Source:** load from Fontshare (`https://api.fontshare.com/v2/css?f[]=general-sans@200,300,400,500,600,700&display=swap`)
- **Registration:** register in `@theme` as `--font-sans: "General Sans", sans-serif;`
- **Pairs with:** Inter, Geist, DM Sans

### Host Grotesk

Uniwidth sans-serif. Letter widths stay the same across all weights, so changing weight never shifts layout. Good for tabs, buttons, navigation, and anywhere a weight change must not cause reflow. Works for headlines and body.

- **Source:** load from Google Fonts (`family=Host+Grotesk:wght@300..800`)
- **Registration:** register in `@theme` as `--font-sans: "Host Grotesk", sans-serif;`
- **Pairs with:** Inter, Geist, DM Sans

### Instrument Sans

Geometric neo-grotesque built from straight lines and simple circles. Uniform strokes, straight terminals. Has 12 stylistic sets for alternate glyphs. Best for clean, technical interfaces. Works for headlines and body.

- **Weight restriction:** only supports `font-normal` (400). Never use `font-medium`, `font-semibold`, or `font-bold`
- **Source:** load from Google Fonts (`family=Instrument+Sans:wght@400..700`)
- **Registration:** register in `@theme` as `--font-sans: "Instrument Sans", sans-serif;`
- **Pairs with:** Inter, Geist, DM Sans

### Instrument Serif

High-contrast editorial serif for headlines and display text. With a clean sans-serif body font it gives pages a premium, editorial feel. Works well for marketing sites, landing pages, and brand-forward designs.

- **Sizing:** Instrument Serif is optically small. Never use `text-4xl` or smaller for headings. Use `text-5xl` and up where other fonts would use `text-4xl`
- **Source:** load from Google Fonts
- **Registration:** register in `@theme` as `--font-display: "Instrument Serif", serif;`
- **Pairs with:** Inter, Geist, DM Sans

### Inter

Clean, highly legible sans-serif designed for screens. Works for body copy, app UI, and headings.

- **Source:** always load from `https://rsms.me/inter/inter.css` or self-host. Never use the Google Fonts version. It lacks the Display optical size variant and `font-feature-settings` support
- **Optical sizing:** Inter includes a Display variant that kicks in automatically at larger sizes via `font-optical-sizing: auto`. The Google Fonts build strips this out
- **Feature settings:** turn on optional OpenType features to give Inter a more custom feel: `cv02` (single-story `a`), `cv03` (open `6`/`9`), `cv04` (open `4`), `cv11` (single-story `l`), `ss01` (open digits), `ss03` (round quotes)
- **Registration:** register in `@theme` as `--font-sans: "InterVariable", sans-serif;` with `--font-sans--font-feature-settings: "cv02", "cv03", "cv04", "cv11";` to turn features on globally
- **Pairs with:** Geist, DM Sans

### Lora

Well-balanced contemporary serif with roots in calligraphy. Moderate contrast with subtle brush-stroke terminals. Readable at body sizes and still refined. Works for editorial content, blogs, long-form reading, and headlines.

- **Source:** load from Google Fonts (`family=Lora:wght@400..700`)
- **Registration:** register in `@theme` as `--font-serif: "Lora", serif;`
- **Pairs with:** Inter, Geist, DM Sans, Satoshi

### Mona Sans

GitHub's neo-grotesque, with an optical size axis that adjusts letterforms automatically at different sizes. Strong, industrial feel. Works for headlines and body.

- **Source:** load from Google Fonts (`family=Mona+Sans:wght@200..900`)
- **Width axis:** has a `wdth` variable axis. Use a wider value (e.g. `"wdth" 112.5`) for headlines to give them a bolder, more expanded feel. The wide variant is strictly for headlines, never body copy
- **Registration:** register in `@theme` as `--font-sans: "Mona Sans", sans-serif;`. When using the wide variant for headlines, also register `--font-display: "Mona Sans", sans-serif;` with `--font-display--font-variation-settings: "wdth" 112.5;`
- **Pairs with:** Inter, Geist, DM Sans

### Satoshi

Modernist sans-serif that blends rounded shapes with sharp angular details. The double-storey `a` and `g` give it more personality than most geometrics. Lean into that for brand-forward designs. Works for headlines and body.

- **Source:** load from Fontshare (`https://api.fontshare.com/v2/css?f[]=satoshi@300,400,500,700,900&display=swap`)
- **Registration:** register in `@theme` as `--font-sans: "Satoshi", sans-serif;`
- **Pairs with:** Inter, Geist, DM Sans
