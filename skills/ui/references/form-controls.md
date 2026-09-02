# Form controls

Rules for inputs, selects, textareas, checkboxes, radio buttons, toggles, search bars, checkout forms, auth forms, and input/button combos.

## Design rules

- Never pair `shadow-*` with solid gray borders on any form control:

  Don't:

  ```html
  <... class="border border-gray-300 shadow-* ..." />
  ```

  ```html
  <... class="border border-gray-950/10 shadow-* ..." />
  ```

  Do:

  ```html
  <... class="ring-1 ring-black/10 shadow-* ..." />
  ```

- Use `max-w-xs` for compact, single-purpose forms like login, sign-up, or single-field inputs. `max-w-sm` and wider is too roomy for focused UI
- If a text input's font size is smaller than `16px`, add `max-sm:text-base/{lh}` to bump it to `16px` on mobile
- Never use `outline-offset-*` on custom focus rings for `<input>` and `<textarea>` elements. Use `outline-offset-0` or leave the offset out
- When using a 2px focus outline on `<input>` or `<textarea>`, inset it with `-outline-offset-1` so it doesn't stick out past the element
- Never use the conjoined input + button pattern where they share a border. Put a gap between them or nest the button visually inside the input

## Coding rules

- Always include a `name` attribute on `<input>`, `<select>`, and `<textarea>` elements
- Every `<input>`, `<select>`, and `<textarea>` needs either a matching `<label>` linked via `id`/`for`, or an `aria-label` attribute
- Always set an explicit `type` attribute on `<button>` elements: `type="submit"` inside forms, `type="button"` otherwise
- Use `placeholder` with `aria-label` instead of visible `<label>` elements for ecommerce/checkout forms where the field's purpose is obvious from context. Still use section headings (e.g. "Shipping address", "Payment") to group related fields

## Selects

- Use a custom chevron so styling matches across browsers. Wrap only the `<select>` and chevron in `inline-grid grid-cols-[1fr_--spacing(8)]` (never the label). Add `col-span-full row-start-1 appearance-none pr-8` to the `<select>`. Place an SVG chevron with `pointer-events-none col-start-2 row-start-1 place-self-center`

```html
<svg
  viewBox="0 0 8 5"
  width="8"
  height="5"
  fill="none"
  class="pointer-events-none col-start-2 row-start-1 place-self-center"
>
  <path d="M.5.5 4 4 7.5.5" stroke="currentcolor" />
</svg>
```

## Checkboxes

- Use a native `<input type="checkbox">`
- All the styles come from CSS based on the input state
- **Never use JavaScript to toggle classes based on input state.** Use CSS states and variants only
- Replace `{brand}` with the right brand color
- Every class is required. Don't drop any
- When there's a label, link it to the input with `id` and `for`. Otherwise give the input an `aria-label`
- To vertically center a checkbox with text next to it, wrap it in an element with `h-lh items-center` and the matching `text-{size}`. Never put `h-lh` on the `inline-grid` wrapper itself. Never use top margins or manual alignment
- Checkboxes should be larger on mobile, e.g. `size-5 sm:size-4`

```html
<span class="group inline-grid size-4 grid-cols-1">
  <input
    type="checkbox"
    class="checked:border-{brand} checked:bg-{brand} indeterminate:border-{brand} indeterminate:bg-{brand} focus-visible:outline-{brand} dark:checked:border-{brand} dark:checked:bg-{brand} dark:indeterminate:border-{brand} dark:indeterminate:bg-{brand} dark:focus-visible:outline-{brand} col-start-1 row-start-1 appearance-none rounded-sm border border-gray-300 bg-white focus-visible:outline-2 focus-visible:outline-offset-2 disabled:border-gray-300 disabled:bg-gray-100 disabled:checked:bg-gray-100 dark:border-white/10 dark:bg-white/5 dark:disabled:border-white/5 dark:disabled:bg-white/10 dark:disabled:checked:bg-white/10 forced-colors:appearance-auto"
  />
  <svg
    viewBox="0 0 14 14"
    fill="none"
    class="pointer-events-none col-start-1 row-start-1 size-7/8 self-center justify-self-center stroke-white group-has-disabled:stroke-gray-950/25 dark:group-has-disabled:stroke-white/25"
  >
    <path
      d="M3 8L6 11L11 3.5"
      stroke-width="2"
      stroke-linecap="round"
      stroke-linejoin="round"
      class="group-not-has-checked:opacity-0"
    />
    <path
      d="M3 7H11"
      stroke-width="2"
      stroke-linecap="round"
      stroke-linejoin="round"
      class="group-not-has-indeterminate:opacity-0"
    />
  </svg>
</span>
```

## Radio buttons

- Use a native `<input type="radio">`
- All the styles come from CSS based on the input state
- **Never use JavaScript to toggle classes based on input state.** Use CSS states and variants only
- Replace `{brand}` with the right brand color
- Every class is required. Don't drop any
- When there's a label, link it to the input with `id` and `for`. Otherwise give the input an `aria-label`
- To vertically center a radio button with text next to it, wrap it in an element with `h-lh items-center` and the matching `text-{size}`. Never put `h-lh` on the `inline-grid` wrapper itself. Never use top margins or manual alignment
- Radio buttons should be larger on mobile, e.g. `size-5 sm:size-4`

```html
<span class="group inline-grid size-4 grid-cols-1">
  <input
    type="radio"
    class="checked:border-{brand} checked:bg-{brand} focus-visible:outline-{brand} dark:checked:border-{brand} dark:checked:bg-{brand} dark:focus-visible:outline-{brand} col-start-1 row-start-1 appearance-none rounded-full border border-gray-300 bg-white focus-visible:outline-2 focus-visible:outline-offset-2 disabled:border-gray-300 disabled:bg-gray-100 disabled:checked:bg-gray-100 dark:border-white/10 dark:bg-white/5 dark:disabled:border-white/5 dark:disabled:bg-white/10 dark:disabled:checked:bg-white/10 forced-colors:appearance-auto"
  />
  <span
    class="pointer-events-none col-start-1 row-start-1 size-[round(down,40%,1px)] self-center justify-self-center rounded-full bg-white group-not-has-checked:opacity-0 group-has-disabled:bg-gray-400 dark:group-has-disabled:bg-white/25"
  ></span>
</span>
```

## Toggles

- Use a native `<input type="checkbox">`
- All the styles come from CSS based on the input state
- **Never use JavaScript to toggle classes based on input state.** Use CSS states and variants only
- Replace `{brand}` with the right brand color
- Replace `{gray}` with the right gray color
- Every class is required. Don't drop any
- Use `w-9` as the default size. Only change the width to make it larger or smaller
- Toggles should be larger on mobile, e.g. `w-11 sm:w-9`
- When there's a label, link it to the input with `id` and `for`. Otherwise give the input an `aria-label`
- Remove all `dark:` classes if the site doesn't support dark mode. For always-dark sites, use only the `dark:` variant values as the base classes and drop the `dark:` prefixed versions

```html
<div
  class="group outline-{brand}-600 has-checked:bg-{brand}-600 dark:outline-{brand}-500 dark:has-checked:bg-{brand}-500 bg-{gray}-200 inset-ring-{gray}-900/5 relative inline-flex w-9 shrink-0 rounded-full p-0.5 inset-ring outline-offset-2 transition-colors duration-200 ease-in-out has-focus-visible:outline-2 dark:bg-white/5 dark:inset-ring-white/10"
>
  <span
    class="ring-{gray}-900/5 aspect-square w-1/2 rounded-full bg-white ring-1 shadow-xs transition-transform duration-200 ease-in-out group-has-checked:translate-x-full"
  ></span>
  <input type="checkbox" class="absolute inset-0 size-full appearance-none focus:outline-hidden" />
</div>
```
