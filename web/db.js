/**
 * db.js — Mavuno IndexedDB Layer
 * Local-first data storage for offline-capable farmer app.
 *
 * Stores:
 *  - produce      : farmer's produce inventory (versioned)
 *  - listings     : market listings (versioned)
 *  - learning     : cached learning content (read-only offline)
 *  - syncQueue    : pending mutations to flush when online
 */

const DB_NAME = 'mavuno-db';
const DB_VERSION = 1;

/** @type {IDBDatabase|null} */
let _db = null;

/**
 * Open (or upgrade) the IndexedDB database.
 * Returns a Promise<IDBDatabase>.
 */
export function openDB() {
  if (_db) return Promise.resolve(_db);

  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, DB_VERSION);

    req.onupgradeneeded = (event) => {
      const db = event.target.result;

      // ── Produce store ────────────────────────────────────────────────────────
      if (!db.objectStoreNames.contains('produce')) {
        const produceStore = db.createObjectStore('produce', { keyPath: 'id' });
        produceStore.createIndex('syncStatus', 'syncStatus', { unique: false });
        produceStore.createIndex('updatedAt', 'updatedAt', { unique: false });
      }

      // ── Listings store ───────────────────────────────────────────────────────
      if (!db.objectStoreNames.contains('listings')) {
        const listingStore = db.createObjectStore('listings', { keyPath: 'id' });
        listingStore.createIndex('produceId', 'produceId', { unique: false });
        listingStore.createIndex('syncStatus', 'syncStatus', { unique: false });
      }

      // ── Learning content store ───────────────────────────────────────────────
      if (!db.objectStoreNames.contains('learning')) {
        const learningStore = db.createObjectStore('learning', { keyPath: 'id' });
        learningStore.createIndex('category', 'category', { unique: false });
      }

      // ── Sync queue store ─────────────────────────────────────────────────────
      if (!db.objectStoreNames.contains('syncQueue')) {
        const syncStore = db.createObjectStore('syncQueue', {
          keyPath: 'id',
          autoIncrement: true,
        });
        syncStore.createIndex('status', 'status', { unique: false });
        syncStore.createIndex('entityType', 'entityType', { unique: false });
        syncStore.createIndex('createdAt', 'createdAt', { unique: false });
      }
    };

    req.onsuccess = (event) => {
      _db = event.target.result;
      resolve(_db);
    };

    req.onerror = () => reject(req.error);
  });
}

/** Helper — run a transaction and return a Promise. */
function tx(storeName, mode, fn) {
  return openDB().then((db) => {
    return new Promise((resolve, reject) => {
      const transaction = db.transaction(storeName, mode);
      const store = transaction.objectStore(storeName);
      const req = fn(store);
      if (req) {
        req.onsuccess = () => resolve(req.result);
        req.onerror = () => reject(req.error);
      } else {
        transaction.oncomplete = () => resolve();
        transaction.onerror = () => reject(transaction.error);
      }
    });
  });
}

/** Helper — get all records from a store. */
function getAll(storeName) {
  return openDB().then((db) => {
    return new Promise((resolve, reject) => {
      const t = db.transaction(storeName, 'readonly');
      const req = t.objectStore(storeName).getAll();
      req.onsuccess = () => resolve(req.result);
      req.onerror = () => reject(req.error);
    });
  });
}

// ══════════════════════════════════════════════════════════════════════════════
//  PRODUCE  (versioned, local-first)
// ══════════════════════════════════════════════════════════════════════════════

/**
 * Produce record shape:
 * {
 *   id        : string  (client-generated UUID)
 *   name      : string
 *   category  : string  (e.g. "Grain", "Vegetable", "Fruit")
 *   quantity  : number
 *   unit      : string  (e.g. "kg", "bags", "crates")
 *   price     : number  (per unit, KES)
 *   location  : string
 *   notes     : string
 *   version   : number  (for conflict detection)
 *   syncStatus: "pending" | "synced" | "conflict"
 *   createdAt : ISO string
 *   updatedAt : ISO string
 * }
 */

export function getAllProduce() {
  return getAll('produce');
}

export function getProduceById(id) {
  return tx('produce', 'readonly', (store) => store.get(id));
}

export function saveProduce(produce) {
  // Local write — never blocks on network
  const record = {
    ...produce,
    updatedAt: new Date().toISOString(),
    syncStatus: 'pending',
  };
  return tx('produce', 'readwrite', (store) => store.put(record)).then(() => record);
}

export function deleteProduce(id) {
  return tx('produce', 'readwrite', (store) => store.delete(id));
}

export function markProduce(id, syncStatus, serverVersion) {
  return tx('produce', 'readwrite', (store) => {
    const req = store.get(id);
    req.onsuccess = () => {
      if (req.result) {
        req.result.syncStatus = syncStatus;
        if (serverVersion !== undefined) req.result.version = serverVersion;
        store.put(req.result);
      }
    };
    return req;
  });
}

// ══════════════════════════════════════════════════════════════════════════════
//  LISTINGS  (versioned, local-first)
// ══════════════════════════════════════════════════════════════════════════════

/**
 * Listing record shape:
 * {
 *   id         : string
 *   produceId  : string
 *   produceName: string  (denormalised for offline display)
 *   quantity   : number
 *   price      : number
 *   location   : string
 *   contact    : string
 *   status     : "available" | "sold" | "expired"
 *   version    : number
 *   syncStatus : "pending" | "synced" | "conflict"
 *   createdAt  : ISO string
 *   updatedAt  : ISO string
 * }
 */

export function getAllListings() {
  return getAll('listings');
}

export function getListingById(id) {
  return tx('listings', 'readonly', (store) => store.get(id));
}

export function saveListing(listing) {
  const record = {
    ...listing,
    updatedAt: new Date().toISOString(),
    syncStatus: 'pending',
  };
  return tx('listings', 'readwrite', (store) => store.put(record)).then(() => record);
}

export function deleteListing(id) {
  return tx('listings', 'readwrite', (store) => store.delete(id));
}

// ══════════════════════════════════════════════════════════════════════════════
//  LEARNING  (cached content)
// ══════════════════════════════════════════════════════════════════════════════

/**
 * Learning record shape:
 * {
 *   id       : string
 *   title    : string
 *   category : string
 *   body     : string  (markdown or plain text)
 *   tags     : string[]
 *   cachedAt : ISO string
 * }
 */

export function getAllLearning() {
  return getAll('learning');
}

export function saveLearningContent(articles) {
  return openDB().then((db) => {
    return new Promise((resolve, reject) => {
      const t = db.transaction('learning', 'readwrite');
      const store = t.objectStore('learning');
      articles.forEach((a) => store.put({ ...a, cachedAt: new Date().toISOString() }));
      t.oncomplete = () => resolve();
      t.onerror = () => reject(t.error);
    });
  });
}

// ══════════════════════════════════════════════════════════════════════════════
//  SYNC QUEUE
// ══════════════════════════════════════════════════════════════════════════════

/**
 * SyncQueue record shape:
 * {
 *   id         : number  (auto-increment)
 *   entityType : "produce" | "listing"
 *   operation  : "create" | "update" | "delete"
 *   entityId   : string
 *   payload    : object
 *   retryCount : number
 *   status     : "pending" | "processing" | "failed"
 *   createdAt  : ISO string
 * }
 */

export function enqueueSyncOp(op) {
  const record = {
    ...op,
    retryCount: 0,
    status: 'pending',
    createdAt: new Date().toISOString(),
  };
  return tx('syncQueue', 'readwrite', (store) => store.add(record));
}

export function getPendingSyncOps() {
  return openDB().then((db) => {
    return new Promise((resolve, reject) => {
      const t = db.transaction('syncQueue', 'readonly');
      const index = t.objectStore('syncQueue').index('status');
      const req = index.getAll('pending');
      req.onsuccess = () => resolve(req.result);
      req.onerror = () => reject(req.error);
    });
  });
}

export function updateSyncOp(id, changes) {
  return openDB().then((db) => {
    return new Promise((resolve, reject) => {
      const t = db.transaction('syncQueue', 'readwrite');
      const store = t.objectStore('syncQueue');
      const req = store.get(id);
      req.onsuccess = () => {
        if (req.result) {
          store.put({ ...req.result, ...changes });
        }
        resolve();
      };
      req.onerror = () => reject(req.error);
    });
  });
}

export function removeSyncOp(id) {
  return tx('syncQueue', 'readwrite', (store) => store.delete(id));
}

export function getSyncQueueCount() {
  return openDB().then((db) => {
    return new Promise((resolve, reject) => {
      const t = db.transaction('syncQueue', 'readonly');
      const index = t.objectStore('syncQueue').index('status');
      const req = index.count('pending');
      req.onsuccess = () => resolve(req.result);
      req.onerror = () => reject(req.error);
    });
  });
}

/** Seed default learning content if store is empty. */
export async function seedLearningIfEmpty() {
  const existing = await getAllLearning();
  if (existing.length > 0) return;

  const defaultContent = [
    {
      id: 'learn-001',
      title: 'Soil Preparation for Maize',
      category: 'Crop Management',
      body: `Good soil preparation is the foundation of a successful maize harvest.

**Steps:**
1. Clear the field of weeds and crop residue
2. Deep plough to 20–30 cm depth
3. Harrow to break clods and create fine seedbed
4. Test soil pH (ideal: 5.5–7.0)
5. Apply lime if pH is too low

**Tip:** Rotate with legumes every 2 seasons to restore nitrogen.`,
      tags: ['maize', 'soil', 'planting'],
    },
    {
      id: 'learn-002',
      title: 'Pest Management: Stemborer',
      category: 'Pest Control',
      body: `Stemborers are one of the biggest threats to cereal crops in East Africa.

**Signs of infestation:**
- Dead heart in young plants
- Dry leaves (window pane damage)
- Holes in stems

**Management:**
- Early planting avoids peak moth populations
- Use push-pull intercropping (Desmodium + Napier grass)
- Apply Bt-based biopesticides
- Spray in the whorl stage

**Note:** Avoid burning crop residue — it destroys natural enemies.`,
      tags: ['pest', 'maize', 'stemborer', 'organic'],
    },
    {
      id: 'learn-003',
      title: 'Post-Harvest Storage Best Practices',
      category: 'Post-Harvest',
      body: `Proper storage prevents aflatoxin and reduces post-harvest losses.

**Checklist:**
- Dry grain to <13% moisture before storage
- Use hermetic bags (PICS bags) for grain
- Inspect bags monthly for pests
- Store in a cool, dry, ventilated room
- Keep bags off the ground on pallets

**Warning:** Never mix new and old grain in the same storage unit.`,
      tags: ['storage', 'grain', 'post-harvest'],
    },
    {
      id: 'learn-004',
      title: 'How to Price Your Produce',
      category: 'Market Skills',
      body: `Pricing your produce correctly means more income per season.

**Formula:**
Price = Cost of Production + Profit Margin + Market Premium

**Cost of production includes:**
- Seeds, fertiliser, pesticides
- Labour (including your own)
- Transport to market

**Tips:**
- Check prevailing market prices at 3 markets before deciding
- Factor in transport costs to buyer
- Collective pricing through farmer groups gives more negotiating power

**Common mistake:** Selling at any price after harvest due to storage fears.
**Solution:** Use proper storage to sell when prices improve.`,
      tags: ['market', 'pricing', 'income'],
    },
    {
      id: 'learn-005',
      title: 'Water-Smart Farming Techniques',
      category: 'Climate Adaptation',
      body: `Unpredictable rainfall requires smart water management approaches.

**Rainwater Harvesting:**
- Dig shallow trenches (zai pits) to capture runoff
- Use half-moon catchments on slopes
- Install roof catchment systems for homestead gardens

**Conservation Agriculture:**
- Minimum tillage reduces moisture loss
- Mulching keeps soil moist longer
- Cover crops reduce evaporation

**Irrigation Options (smallholder-friendly):**
- Drip kits (affordable for small plots)
- Gravity-fed systems from elevated tanks

**Bonus:** Drought-tolerant varieties of maize and sorghum reduce risk.`,
      tags: ['water', 'climate', 'irrigation', 'drought'],
    },
  ];

  await saveLearningContent(defaultContent);
}

/** Generate a UUID v4 (for client side). */
export function generateId() {
  if (crypto && crypto.randomUUID) return crypto.randomUUID();
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    return (c === 'x' ? r : (r & 0x3) | 0x8).toString(16);
  });
}
