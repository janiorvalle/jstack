# Heading groups

Rules for the headline, subheadline, and optional eyebrow at the top of a marketing or landing page section.

A heading group is a headline and subheadline (plus an optional eyebrow) at the top of a marketing or landing page section. Think of the title and description above a feature grid, team grid, pricing table, testimonial section, CTA, or hero. These rules are for promotional and marketing sections only. They don't apply to blog posts, articles, docs, or editorial content.

- Never constrain the width of a heading group wrapper. No `max-w-*`, no `max-lg:max-w-*`, no width constraints of any kind on the wrapper `<div>`. Always constrain each text element (headline, subheadline) on its own with `max-w-[*ch]` right on the element: `text-base` gets `max-w-[56ch]`, `text-lg` gets `max-w-[48ch]`, `text-xl` gets `max-w-[40ch]`, `text-2xl` to `text-3xl` get `max-w-[40ch]`, `text-4xl` gets `max-w-[35ch]`, `text-5xl` gets `max-w-[30ch]`, `text-6xl` gets `max-w-[24ch]`, `text-7xl` gets `max-w-[20ch]`.

  Example:

  ```html
  <div class="/* never add a max width here */">
    <h2 class="mx-auto max-w-[35ch] text-4xl font-semibold tracking-tight text-balance">…</h2>
    <p class="mx-auto mt-6 max-w-[48ch] text-lg text-pretty text-gray-600">…</p>
  </div>
  ```

- Always use a left-aligned layout for heading groups when the subheadline is longer than ~120 characters (~3 lines when centered)
  - **Ask the user** if a centered layout is requested but the subheadline is longer than ~120 characters. Offer a rewritten version that fits. Only center if the user accepts the shorter copy
