package site

const pageKey = "/"

const articleHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="light">
<title>FastyGo Instant</title>
<style>
:root{color-scheme:light;--bg:#fffaf2;--ink:#21170f;--muted:#6d5a49;--line:#eadbc8;--accent:#9b4a1a}
*{box-sizing:border-box}
html{background:var(--bg);font-size:18px}
body{margin:0;color:var(--ink);font-family:Georgia,"Times New Roman",serif;line-height:1.64;text-rendering:optimizeLegibility}
main{width:min(100% - 2rem,42rem);margin:0 auto;padding:3rem 0 4rem}
header{border-bottom:1px solid var(--line);margin:0 0 2rem;padding:0 0 1.5rem}
p,ul{margin:0 0 1.2rem}
h1{font-size:clamp(2.1rem,8vw,4.2rem);line-height:.96;letter-spacing:-.055em;margin:0 0 1rem}
h2{font-size:1.45rem;line-height:1.15;letter-spacing:-.025em;margin:2rem 0 .75rem}
.eyebrow{color:var(--accent);font:700 .74rem/1.2 ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;letter-spacing:.11em;margin:0 0 1rem;text-transform:uppercase}
.lead{color:var(--muted);font-size:1.18rem;line-height:1.5;margin:0}
.note{border-left:3px solid var(--accent);color:var(--muted);padding:.25rem 0 .25rem 1rem}
code{background:#f2e6d6;border:1px solid var(--line);border-radius:.35rem;font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;font-size:.82em;padding:.1rem .3rem}
ul{padding-left:1.15rem}
li{margin:.35rem 0}
footer{border-top:1px solid var(--line);color:var(--muted);font-size:.88rem;margin-top:2.5rem;padding-top:1.25rem}
@media (max-width:520px){html{font-size:17px}main{width:min(100% - 1.25rem,42rem);padding:2rem 0 3rem}}
</style>
</head>
<body>
<main>
<header>
<p class="eyebrow">FastyGo Instant Example</p>
<h1>A tiny page designed to feel already loaded.</h1>
<p class="lead">This example demonstrates the fastest useful shape of a FastyGo web surface: one prebuilt HTML document, inline CSS, no JavaScript, no external files, no images, no font requests, and no runtime template work on the request path.</p>
</header>
<article>
<p>The goal is not to hide latency with animation. The goal is to remove every piece of avoidable work before the browser paints. A mobile WebView opened from a Telegram Mini App, a messenger link, or an in-app browser should receive a complete readable document in the first response.</p>
<h2>What the server does</h2>
<p>At startup, the application builds a fixed <code>instant.Store</code> with explicit page and byte budgets. The article body is copied once, assigned an ETag, and kept immutable for the lifetime of the process. Requests only select the prebuilt page, set a small set of HTTP headers, and write bytes.</p>
<ul>
<li>No localization negotiation.</li>
<li>No theme switching.</li>
<li>No asset pipeline.</li>
<li>No static directory.</li>
<li>No runtime markdown, templ, CSS, image, script, or font loading.</li>
</ul>
<h2>Why this is useful</h2>
<p>Most product pages eventually need richer interaction. But many entry points do not. A campaign article, announcement, menu excerpt, onboarding note, policy page, or emergency status page can be rendered as a single stable document. That makes it cheap to cache, easy to benchmark, and predictable under load.</p>
<p class="note">The memory budget is intentionally explicit in the application: one page, a small byte cap, and startup failure if content grows beyond the configured limits.</p>
<h2>How to read the benchmark</h2>
<p>The benchmark measures the handler path after startup work is complete. It does not include network, TLS, kernel scheduling, CDN behavior, or a browser. That keeps the signal focused on the framework-side cost of serving an instant page.</p>
<p>If the page changes, restart the process. There is no background refresh and no hidden cache invalidation loop. Simplicity is the feature.</p>
</article>
<footer>FastyGo Instant ships one HTML response and lets the browser paint immediately.</footer>
</main>
</body>
</html>`
