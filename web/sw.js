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
  '/manifest.json'
];

// API base used by sync logic in the service worker.
const API_BASE = '/api';

// Match the retry policy used in the page sync implementation.
const MAX_RETRIES = 5;

// Install event: pre-cache the application shell.
// Using `addAll` will fail the install if any resource is missing; keep this
// behavior for a strict pre-cache of required assets.
self.addEventListener('install', event => {
  // Cache assets individually so a single missing file doesn't fail the
  // entire install. This is more resilient during development.
  event.waitUntil((async () => {
    const cache = await caches.open(STATIC_CACHE);
    for (const url of APP_CACHE) {
      try {
        await cache.add(url);
      } catch (err) {
        // Log missing or failing assets but continue.
        console.warn('ServiceWorker: failed to cache', url, err && err.message ? err.message : err);
      }
    }
    await self.skipWaiting();
  })());
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

// Helper: open IndexedDB from the service worker context.
function openIDB() {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open('mavuno-db');
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
    // no upgrade handling in SW — DB schema is created by the page
  });
}

// Helper: read pending sync ops from IndexedDB
async function getPendingSyncOpsSW() {
  const db = await openIDB();
  return new Promise((resolve, reject) => {
    try {
      const tx = db.transaction('syncQueue', 'readwrite');
      const store = tx.objectStore('syncQueue');
      let req;
      try {
        const idx = store.index('status');
        req = idx.getAll('pending');
      } catch (e) {
        req = store.getAll();
      }
      req.onsuccess = () => resolve(req.result || []);
      req.onerror = () => reject(req.error);
    } catch (err) { reject(err); }
  });
}

async function removeSyncOpSW(id) {
  const db = await openIDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction('syncQueue', 'readwrite');
    tx.objectStore('syncQueue').delete(id);
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error);
  });
}

async function updateSyncOpSW(id, changes) {
  const db = await openIDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction('syncQueue', 'readwrite');
    const store = tx.objectStore('syncQueue');
    const req = store.get(id);
    req.onsuccess = () => {
      if (req.result) {
        store.put({ ...req.result, ...changes });
      }
      resolve();
    };
    req.onerror = () => reject(req.error);
  });
}

// Process a single queued operation inside the service worker.
async function processOpSW(op) {
  const { id, entityType, operation, entityId, payload } = op;
  let method, path;
  switch (operation) {
    case 'create': method = 'POST'; path = `/${entityType}`; break;
    case 'update': method = 'PUT'; path = `/${entityType}/${entityId}`; break;
    case 'delete': method = 'DELETE'; path = `/${entityType}/${entityId}`; break;
    default:
      await removeSyncOpSW(id);
      return;
  }

  try {
    await updateSyncOpSW(id, { status: 'processing' });
    const res = await fetch(`${API_BASE}${path}`, {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: operation === 'delete' ? null : JSON.stringify(payload),
    });

    if (res.ok) {
      await removeSyncOpSW(id);
      return;
    }

    if (res.status === 409) {
      // conflict — remove from queue and rely on client to handle
      await removeSyncOpSW(id);
      // notify clients about conflict
      const clientsList = await self.clients.matchAll({ includeUncontrolled: true });
      for (const c of clientsList) c.postMessage({ type: 'SYNC_CONFLICT', op });
      return;
    }

    // server error — increment retry count
    const newRetry = (op.retryCount || 0) + 1;
    if (newRetry >= MAX_RETRIES) {
      await updateSyncOpSW(id, { status: 'failed', retryCount: newRetry });
    } else {
      await updateSyncOpSW(id, { status: 'pending', retryCount: newRetry });
    }
  } catch (err) {
    const newRetry = (op.retryCount || 0) + 1;
    if (newRetry >= MAX_RETRIES) {
      await updateSyncOpSW(id, { status: 'failed', retryCount: newRetry });
    } else {
      await updateSyncOpSW(id, { status: 'pending', retryCount: newRetry });
    }
  }
}

// Drain queued operations inside the SW.
async function drainSyncQueueSW() {
  const ops = await getPendingSyncOpsSW();
  if (!ops || ops.length === 0) return;
  for (const op of ops) {
    await processOpSW(op);
  }
}

// Background sync handler: listen for named sync events and perform
// pending background tasks (e.g., flushing locally queued writes to server).
// This is a placeholder that logs when a `sync-todos` tag is fired.
self.addEventListener('sync', event => {
  if (event.tag === 'sync-todos') {
    event.waitUntil((async () => {
      console.log('[SW] Background sync fired: draining queue');
      // Notify any open clients so they can also run their client-side sync
      const clientsList = await self.clients.matchAll({ includeUncontrolled: true });
      for (const c of clientsList) {
        c.postMessage({ type: 'BACKGROUND_SYNC' });
      }
      try {
        await drainSyncQueueSW();
      } catch (err) {
        console.error('[SW] drainSyncQueueSW failed', err);
      }
    })());
  }
});

// Unified fetch handler: handles API reads/writes and static assets in one
// listener to avoid multiple `respondWith` calls which cause runtime errors.
self.addEventListener('fetch', event => {
  const { request } = event;

  // API GET → network-first (update API cache), fallback to cache on failure.
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
    return;
  }

  // API write operations → try network; if offline/fails return queued response.
  if (
    request.url.includes('/api/') &&
    (request.method === 'POST' || request.method === 'PUT' || request.method === 'DELETE')
  ) {
    event.respondWith(
      fetch(request.clone())
        .catch(() => new Response(JSON.stringify({ status: 'queued-offline' }), {
          status: 202,
          headers: { 'Content-Type': 'application/json' }
        }))
    );
    return;
  }

  // Static assets → cache-first, fall back to network, then to root '/'.
  event.respondWith(
    caches.match(request).then(cachedResponse => {
      return cachedResponse || fetch(request).catch(() => caches.match('/'));
    })
  );
});
