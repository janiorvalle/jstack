# Testimonials

These rules apply to customer quotes, reviews, social proof sections, testimonial cards, quote punctuation, avatars, and attribution.

## Design rules

- Use hanging punctuation on quotes: `relative before:absolute before:inline before:-translate-x-full before:content-['\201C'] after:inline after:content-['\201D']`
- Always bottom-align the avatars and names across equal-height testimonial cards. Put `flex flex-col justify-between` on each card, and wrap the quote and the attribution in their own elements.
- Don't add whitespace around the quote inside `<p>` tags. Write `<p>The quote text</p>`, not `<p> The quote text </p>`. The extra space breaks the hanging punctuation.
- Follow avatars.md and placeholder-content.md for testimonial photos.
- Use unisex names. The avatars are random, so the name has to work with any photo.
