# General

Markup and Tailwind CSS rules that apply everywhere, not to one component.

## Coding rules

### Markup

- Never put `text-*` (font size) or `leading-*` (line height) classes on inline elements like `<span>`, `<a>`, `<strong>`, `<em>`, or `<code>`. Always put them on the containing block-level element, like `<div>`, `<p>`, `<h1>` to `<h6>`, `<li>`, or `<td>`
- Never add display classes that match an element's default display. So no `block` on `<div>`, `<p>`, `<h1>` to `<h6>`. No `inline` on `<span>`, `<a>`. No `inline-block` on `<input>`, `<button>`, `<select>`. No `table` on `<table>`. This only applies to classes that don't change child layout. `flex`, `grid`, `inline-flex`, `inline-grid` are never redundant
- Never put conflicting classes for the same property on the same element without a variant to tell them apart. No `outline-1 outline-2`, no `outline-black/5 outline-white`. Keep only the value you mean
- Always add `role="list"` to `<ul>` and `<ol>` elements unless a `list-style-*` class (e.g. `list-disc`, `list-decimal`) is applied

### Tailwind CSS

- Always apply `antialiased` to the root element
- Always apply `isolate` to the main app container (the element that gets `inert` when dialogs open). It stops z-index fights with portalled elements
- Put `@import` statements with remote URLs (`http`/`https`) or `url()` at the very top of the CSS file, before `@import "tailwindcss"`, but after `@charset` if there is one
- Add `tabular-nums` to elements that show numbers, especially values that change over time (e.g. counters, timers, prices, stats). It stops layout shift as digits update
- Never use `mt-*`/`mb-*`/`ml-*`/`mr-*`/`mx-*`/`my-*` between flex/grid children. Use `gap-*` on the parent instead
- Prefer `size-{n}` over `h-{n} w-{n}` when both values are the same
- Prefer shorthand classes over split axis classes. `p-8` not `px-8 py-8`, `inset-0` not `inset-x-0 inset-y-0`. Keep them split when a variant overrides one axis, e.g. `p-8 md:px-10`
- Use `--spacing(…)` for arbitrary spacing values. `--padding: --spacing(2)` not `--padding: 8px`
- Never use `calc(var(--spacing)*…)`. Use `--spacing(…)` instead
- Never use `theme(spacing.…)`. Use `--spacing(…)` instead
- Never use `theme()` for colors or other tokens in arbitrary values. Use CSS variables instead. `[stop-color:var(--color-emerald-500)]` not `[stop-color:theme(colors.emerald.500)]`
- Use `rem` for arbitrary font sizes. `text-[0.8125rem]` not `text-[13px]`
- Pixels are fine for properties where Tailwind uses pixels natively, e.g. `border-*`, `outline-*`
- Use theme variable references for arbitrary radii. `--radius: var(--radius-xl)` not `--radius: 16px`
- Never use named line-height values like `tight`, `snug`, `relaxed`. Not in `leading-tight`, not in `text-6xl/tight`. Only use spacing scale values (e.g. `leading-6`, `text-sm/5`), and only when a custom line height is really needed
- Never use inline `style` attributes for static CSS properties that have no utility class. Use arbitrary property syntax instead. `class="[animation-delay:300ms]"` not `style="animation-delay: 300ms"`
- Set CSS variables with arbitrary property syntax, not inline styles. `class="[--padding:--spacing(3)]"` not `style="--padding: --spacing(3)"` (unless the value is dynamic)
- For dynamic values, prefer CSS variables over setting CSS properties directly in `style` attributes. `class="w-(--progress)" style="--progress: 72%"` not `style="width: 72%"`. Name the variable for what it means in context
- Prefer bare values over arbitrary values for integers and multiples of `0.25`. `z-999` not `z-[999]`
- Prefer bare opacity modifiers on color utilities. `bg-neutral-950/2` not `bg-neutral-950/[0.02]`. Use `[…]` only for values that aren't `0.25` increments
- Negate `hidden` with a single conditional variant instead of setting `hidden` and re-applying the display class conditionally. `flex items-center gap-x-6 max-lg:hidden` not `hidden lg:flex lg:items-center lg:gap-x-6`. `not-dark:hidden` not `hidden dark:block`
- Prefer `not-*` variants over setting a base value and overriding it conditionally. `group-not-has-checked:opacity-0` not `group-has-checked:opacity-100 opacity-0`. Put `not-` right before the state you're negating. Not `not-group-has-checked:…` (would fire without a `group` parent) and not `group-has-not-checked:…` (would match any unchecked element)
- Use bare values in variants over arbitrary values in variants. `data-closed:…` not `data-[closed]:…`, `group-data-open:…` not `group-data-[open]:…`
- Always use classes like `min-h-dvh/svh/lvh`, never `min-h-screen` (`screen` is deprecated)
- Always use `bg-linear-*` for gradients, never `bg-gradient-*` (deprecated)
- Use `shrink-*` not `flex-shrink-*`, `grow-*` not `flex-grow-*` (deprecated)
- Prefer whole-number ratios in arbitrary grid/flex values. `grid-cols-[21fr_19fr]` not `grid-cols-[1.05fr_0.95fr]`. Multiply every value by the same factor to get rid of decimals
- Prefer `@utility my-utility { … }` over plain class selectors (`.my-utility { … }`) for reusable styles. Utilities work with all Tailwind variants (`hover:my-utility`, `lg:my-utility`)
- Use `@utility my-utility-* { … }` with `--value()` and `--modifier()` for parameterized utilities that take arguments
- Use `@variant the-variant { … }` inside `@utility` definitions to apply an existing variant. Don't hand-write the media query or selector
- Use `@custom-variant` to define new custom variants when the built-in set doesn't cover the case
- Never nest `@utility` inside another at-rule like `@media` or `@supports`. Move the at-rule inside the `@utility` block instead
