// Cache name for static application shell (HTML/CSS/JS).
const STATIC_CACHE = 'static-v1';

// Cache name for API responses (kept separate from static assets).
const API_CACHE = 'api-v1';

// Application shell files to pre-cache during install so the app can load offline.
const APP_CACHE = [
  '/',
  '/index.html',
  '/styles.css',
  '/app.js',
  '/db.js',
  '/manifest.json',
  '/icons/icon-192.png',
  '/icons/icon-512.png'
];

// Install event: pre-cache the application shell.
// Using `addAll` will fail the install if any resource is missing; keep this
// behavior for a strict pre-cache of required assets.
self.addEventListener('install', event => {
  event.waitUntil(
    caches.open(STATIC_CACHE)
      .then(cache => cache.addAll(APP_CACHE))
      .then(() => self.skipWaiting())
  );
});

// Activate event: clean up any old caches that don't match the current names.
// `clients.claim()` makes the worker take control of uncontrolled clients as soon
// as it becomes active.
self.addEventListener('activate', event => {
  event.waitUntil(
    caches.keys().then(cacheNames => {
      return Promise.all(
        cacheNames.filter(name => name !== STATIC_CACHE && name !== API_CACHE)
          .map(name => caches.delete(name))
      );
    }).then(() => self.clients.claim())
  );
});

// Basic fetch handler: network-first for API GETs, cache-first for other assets.
self.addEventListener('fetch', event => {
  const { request } = event;

  if (request.url.includes('/api/') && request.method === 'GET'){
    // Network-first for API GET requests: update cache on success,
    // fall back to cache on network failure.
    event.respondWith(
      caches.open(API_CACHE).then(cache =>
        fetch(request)
          .then(response => {
            if (response.status === 200) {
              cache.put(request, response.clone());
            }
            return response;
          })
          .catch(() => cache.match(request))
      )
    );
  } else {
    // Cache-first for static assets: serve cached response if available,
    // otherwise fetch from network and ultimately fall back to '/'.
    event.respondWith(
        caches.match(request).then(cachedResponse => {
        return cachedResponse || fetch(request).catch(() => caches.match('/'));
        })
    );
  }
});

///

// Background sync handler: listen for named sync events and perform
// pending background tasks (e.g., flushing locally queued writes to server).
// This is a placeholder that logs when a `sync-todos` tag is fired.
self.addEventListener('sync', event => {
  if (event.tag === 'sync-todos') {
    event.waitUntil(
      Promise.resolve().then(() => {
        console.log('[SW] Syncing todos with server...');
      })
    );
  }
});

// Enhanced fetch handler: covers API write operations and static assets.
// - For API POST/PUT/DELETE we attempt a network forward; on failure we
//   return a 202 response indicating the action was queued for offline.
// - Static assets continue to use cache-first strategy.
self.addEventListener('fetch', event => {
  const { request } = event;

  // API GET → handled by the earlier fetch listener (network-first).

  // API POST / PUT / DELETE → try network, return queued response on failure.
  if (
    request.url.includes('/api/') &&
    (request.method === 'POST' ||
     request.method === 'PUT' ||
     request.method === 'DELETE')
  ) {
    event.respondWith(
      fetch(request.clone())
        .catch(() => {
          return new Response(
            JSON.stringify({ status: 'queued-offline' }),
            {
              status: 202,
              headers: { 'Content-Type': 'application/json' }
            }
          );
        })
    );
    return;
  }

  // Static assets → cache-first fallback to network.
  event.respondWith(
    caches.match(request).then(cachedResponse => {
      return cachedResponse ||
           fetch(request).catch(() => caches.match('/'));
    })
  );
});
