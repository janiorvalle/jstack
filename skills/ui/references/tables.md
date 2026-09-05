# Tables

Rules for data tables, comparison tables, table headings, row dividers, tables that scroll sideways, and table containers.

## Design rules

- No uppercase text in table headings. Use sentence case.
- Table headings don't wrap. Put `whitespace-nowrap` on `<th>` elements.
- Don't put tables inside containers or cards. Put them straight on the background.
- Only horizontal lines divide rows. No vertical lines, no outer borders.
- Always use `w-full` so the table fills its container.
- Hide the headings with `sr-only` when the columns explain themselves, usually a table with 2 to 3 columns where headings add nothing.
- If all the columns won't fit on a small screen, make the table responsive with a two-div wrapper:
  - Outer div: `overflow-x-auto whitespace-nowrap` with negative horizontal and vertical margins. The horizontal margins cancel the page container's padding (e.g. `-mx-4 sm:-mx-6 lg:-mx-8`). The vertical margin is always `-my-2`.
  - Inner div: `inline-block min-w-full align-middle` with horizontal padding matching the container's padding (e.g. `px-4 sm:px-6 lg:px-8`) and `py-2`.
  - The negative horizontal margins and the horizontal padding always match the real container padding used in the page layout.
  - Example:
  ```html
  <!-- Example assumes container padding of px-4 sm:px-6 lg:px-8 -->
  <div class="-mx-4 -my-2 overflow-x-auto whitespace-nowrap sm:-mx-6 lg:-mx-8">
    <div class="inline-block min-w-full px-4 py-2 align-middle sm:px-6 lg:px-8">
      <table>
        …
      </table>
    </div>
  </div>
  ```
