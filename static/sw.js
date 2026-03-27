/* ── Service Worker ──────────────────────────────────────────────────────────
 * Minimal cache-on-install strategy for static assets.
 * HTML pages and API calls always go to the network.
 * Bump CACHE version when deploying new static asset versions.
 */

const CACHE = 'sn-v16';

const PRECACHE = [
  '/css/style.css?v=16',
  '/js/app.js?v=16',
  '/js/wallet-picker.js?v=16',
  '/js/qrcode.min.js',
  '/icon-192.png',
  '/icon-512.png',
  '/apple-touch-icon.png',
];

self.addEventListener('install', e => {
  e.waitUntil(
    caches.open(CACHE).then(c => c.addAll(PRECACHE))
  );
  self.skipWaiting();
});

self.addEventListener('activate', e => {
  e.waitUntil(
    caches.keys().then(keys =>
      Promise.all(keys.filter(k => k !== CACHE).map(k => caches.delete(k)))
    )
  );
  self.clients.claim();
});

self.addEventListener('fetch', e => {
  if (e.request.method !== 'GET') return;
  const url = new URL(e.request.url);
  if (url.origin !== self.location.origin) return;

  // Cache-first for precached static assets
  if (PRECACHE.includes(url.pathname)) {
    e.respondWith(
      caches.match(e.request).then(r => r || fetch(e.request))
    );
  }
  // All other requests (HTML, API) go straight to the network
});
