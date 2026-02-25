const STATIC_CACHE = 'static-v1';
const API_CACHE = 'api-v1';

const APP_CACHE = [
  '/',
  '/index.html',
  '/styles.css',
  '/app.js',
  '/favicon.ico',
  '/manifest.json',
  '/sw-test.html',
  '/icons/icon-192.png',
  '/icons/icon-512.png'
];

self.addEventListener('install', event => {
  event.waitUntil((async () => {
    const cache = await caches.open(STATIC_CACHE);
    // Cache assets individually so one missing file doesn't fail the whole install.
    for (const url of APP_CACHE) {
      try {
        await cache.add(url);
      } catch (err) {
        console.warn('ServiceWorker: failed to cache', url, err && err.message ? err.message : err);
      }
    }
    await self.skipWaiting();
  })());
});

self.addEventListener('activate', event => {
  event.waitUntil(
    caches.keys().then(cacheNames => {
      return Promise.all(
        cacheNames.filter(name => name !== STATIC_CACHE && name !== API_CACHE)
          .map(name => caches.delete(name))
      );
    }).then(() => self.clients.claim()) // <-- add this
  );
});

self.addEventListener('fetch', event => {
  const { request } = event;

  if (request.url.includes('/api/') && request.method === 'GET'){
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
    event.respondWith(
        caches.match(request).then(cachedResponse => {
        return cachedResponse || fetch(request).catch(() => caches.match('/'));
        })
    );
  }
});
