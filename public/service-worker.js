// Cache name
const CACHE_NAME = 'snapdrop-cache-v1';

// Files to cache
const urlsToCache = [
  '/',
  '/styles.css',
  '/scripts/network.js',
  '/scripts/ui.js',
  '/scripts/theme.js',
  '/scripts/clipboard.js',
  '/sounds/blop.mp3',
  '/sounds/blop.ogg',
  '/images/favicon-96x96.png'
];

// Install service worker
self.addEventListener('install', event => {
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then(cache => cache.addAll(urlsToCache))
  );
});

// Fetch event handler
self.addEventListener('fetch', event => {
  event.respondWith(
    caches.match(event.request)
      .then(response => response || fetch(event.request))
  );
});
