# Architecture notes

The blog example shows how to combine three reusable framework pieces:

1. `pkg/content-markdown` — a markdown library that pre-renders pages at startup.
2. `pkg/web/locale` — the request-scoped locale negotiator.
3. `pkg/cache` — a sharded TTL cache used to memoize rendered HTML pages.

Every blog feature owns:

- **Routes**: `/`, `/posts/{slug}`.
- **Templ views**: a list page and a single-post page.
- **Content registry**: a slice of `{slug, title}` entries paired with the
  embedded `*.md` files.

Adding new posts is a four-line diff: drop a markdown file, register it,
generate templ, run the server.
