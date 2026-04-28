const SHELL_CACHE = "fastygo-pwa-shell-v1";
const STATIC_CACHE = "fastygo-pwa-static-v1";

const SHELL_URLS = [
  "/",
  "/offline",
  "/labs/pomodoro",
  "/manifest.webmanifest",
  "/static/css/app.css",
  "/static/js/theme.js",
  "/static/js/ui8kit.js",
  "/static/js/pwa-register.js",
  "/static/js/pwa-data.js",
  "/static/js/pwa-pomodoro.js",
  "/static/img/tasks-1.png",
  "/static/img/tasks-2.png",
  "/static/img/tasks-3.png",
  "/static/img/tasks-4.png",
  "/static/img/icons/icon-192.svg",
  "/static/img/icons/icon-512.svg",
  "/static/img/icons/maskable-512.svg"
];

self.addEventListener("install", (event) => {
  event.waitUntil(
    caches.open(SHELL_CACHE)
      .then((cache) => cache.addAll(SHELL_URLS))
      .then(() => self.skipWaiting())
  );
});

self.addEventListener("message", (event) => {
  if (!event.data || event.data.type !== "CLEAR_PWA_CACHE") {
    return;
  }

  event.waitUntil(clearPwaCaches());
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches.keys()
      .then((keys) => Promise.all(
        keys
          .filter((key) => key !== SHELL_CACHE && key !== STATIC_CACHE)
          .map((key) => caches.delete(key))
      ))
      .then(() => self.clients.claim())
  );
});

self.addEventListener("fetch", (event) => {
  const request = event.request;
  if (request.method !== "GET") {
    return;
  }

  const url = new URL(request.url);
  if (request.mode === "navigate") {
    event.respondWith(networkFirstNavigation(request));
    return;
  }

  if (url.pathname.startsWith("/static/") || url.pathname === "/manifest.webmanifest") {
    event.respondWith(cacheFirst(request));
  }
});

async function networkFirstNavigation(request) {
  const cache = await caches.open(SHELL_CACHE);
  try {
    const response = await fetch(request);
    cache.put(request, response.clone());
    return response;
  } catch (_) {
    return await cache.match(request) || await cache.match("/offline");
  }
}

async function cacheFirst(request) {
  const cache = await caches.open(STATIC_CACHE);
  const cached = await cache.match(request);
  if (cached) {
    return cached;
  }

  const response = await fetch(request);
  cache.put(request, response.clone());
  return response;
}

async function clearPwaCaches() {
  const keys = await caches.keys();
  await Promise.all(
    keys
      .filter((key) => key.startsWith("fastygo-pwa-"))
      .map((key) => caches.delete(key))
  );
}
