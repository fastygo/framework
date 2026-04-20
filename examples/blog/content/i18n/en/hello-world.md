# Hello, world

Welcome to the demo blog built on top of **fastygo/framework** and **UI8Kit**.

This post is rendered server-side from a markdown file embedded directly
into the binary. There is no database, no CMS, and no client-side JavaScript
involved in producing the article body — the markdown is converted to HTML
once at startup and cached in memory.

## Why pre-render?

- **Simple operations.** A single Go binary serves both the templ shell
  and the rendered article.
- **Predictable performance.** Rendering happens at boot; requests just
  return cached HTML.
- **Easy versioning.** Posts are tracked in git alongside code.

## What's next?

Drop a new file under `content/i18n/<locale>/<slug>.md`, add a matching entry to the
post registry in `internal/site/blog/feature.go`, and rebuild.
