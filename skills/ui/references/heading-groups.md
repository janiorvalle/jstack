# Heading groups

Rules for the headline, subheadline, and optional eyebrow at the top of a marketing or landing page section: the title and description above a feature grid, team grid, pricing table, testimonial section, CTA, or hero. Promotional and marketing sections only. Not blog posts, articles, docs, or editorial content.

- Never constrain the width of the heading group wrapper. No `max-w-*`, no `max-lg:max-w-*`, no width constraint of any kind on the wrapper `<div>`. Constrain each text element (headline, subheadline) on its own with `max-w-[*ch]` right on the element: `text-base` gets `max-w-[56ch]`, `text-lg` gets `max-w-[48ch]`, `text-xl` gets `max-w-[40ch]`, `text-2xl` to `text-3xl` get `max-w-[40ch]`, `text-4xl` gets `max-w-[35ch]`, `text-5xl` gets `max-w-[30ch]`, `text-6xl` gets `max-w-[24ch]`, `text-7xl` gets `max-w-[20ch]`.

  Example:

  ```html
  <div class="/* never add a max width here */">
    <h2 class="mx-auto max-w-[35ch] text-4xl font-semibold tracking-tight text-balance">…</h2>
    <p class="mx-auto mt-6 max-w-[48ch] text-lg text-pretty text-gray-600">…</p>
  </div>
  ```

- Left-align heading groups when the subheadline is longer than ~120 characters (~3 lines when centered)
  - If a centered layout is requested and the subheadline is longer than ~120 characters, ask the user. Offer a rewritten version that fits. Only center if they accept the shorter copy
