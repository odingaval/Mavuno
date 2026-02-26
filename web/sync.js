/**
 * sync.js — Mavuno Background Sync Engine
 *
 * Responsibilities:
 *  - Watch online/offline transitions
 *  - Drain the syncQueue against the REST API
 *  - Exponential backoff for failed requests
 *  - Handle 409 Conflict responses gracefully
 *  - Emit events the UI can subscribe to
 */

import {
    getPendingSyncOps,
    updateSyncOp,
    removeSyncOp,
    markProduce,
    getSyncQueueCount,
    saveLearningContent,
    saveProduce,
    saveListing,
} from './db.js';

const API_BASE = '/api';

const MAX_RETRIES = 5;
const BASE_BACKOFF_MS = 1000; // 1 second → 2 → 4 → 8 → 16

// ── Event Bus (tiny pub/sub for UI notifications) ─────────────────────────────

const listeners = {};

export function on(event, fn) {
    if (!listeners[event]) listeners[event] = [];
    listeners[event].push(fn);
    return () => {
        listeners[event] = listeners[event].filter((f) => f !== fn);
    };
}

function emit(event, data) {
    (listeners[event] || []).forEach((fn) => fn(data));
}

// ── Sync State ────────────────────────────────────────────────────────────────

let _syncing = false;
let _online = navigator.onLine;

export function isOnline() {
    return _online;
}

export function isSyncing() {
    return _syncing;
}

// ── Network Monitoring ────────────────────────────────────────────────────────

window.addEventListener('online', () => {
    _online = true;
    emit('online');
    pullFromServer().then(() => triggerSync());
});

window.addEventListener('offline', () => {
    _online = false;
    emit('offline');
});

// ── API Helpers ───────────────────────────────────────────────────────────────

async function apiRequest(method, path, body) {
    const opts = {
        method,
        headers: { 'Content-Type': 'application/json' },
    };
    if (body) opts.body = JSON.stringify(body);

    const response = await fetch(`${API_BASE}${path}`, opts);
    return response;
}

// ── Conflict Resolution ───────────────────────────────────────────────────────

async function handleConflict(op, serverData) {
    // Strategy: server wins for now — mark the local record as "conflict"
    // so the UI can surface a notification to the farmer.
    if (op.entityType === 'produce') {
        await markProduce(op.entityId, 'conflict');
    }
    emit('conflict', { op, serverData });
}

// ── Process a Single Sync Operation ──────────────────────────────────────────

async function processSyncOp(op) {
    const { entityType, operation, entityId, payload } = op;

    let method, path;

    switch (operation) {
        case 'create':
            method = 'POST';
            path = `/${entityType}`;
            break;
        case 'update':
            method = 'PUT';
            path = `/${entityType}/${entityId}`;
            break;
        case 'delete':
            method = 'DELETE';
            path = `/${entityType}/${entityId}`;
            break;
        default:
            // Unknown operation — skip
            await removeSyncOp(op.id);
            return;
    }

    try {
        await updateSyncOp(op.id, { status: 'processing' });

        const response = await apiRequest(method, path, operation !== 'delete' ? payload : null);

        if (response.ok) {
            // Success — remove from queue and mark local record as synced
            await removeSyncOp(op.id);
            if (entityType === 'produce') {
                const serverRecord = await response.json().catch(() => null);
                await markProduce(entityId, 'synced', serverRecord?.version);
            }
            emit('synced', { op });
            return;
        }

        if (response.status === 409) {
            const serverData = await response.json().catch(() => null);
            await handleConflict(op, serverData);
            await removeSyncOp(op.id); // Remove — farmer must resolve conflict manually
            return;
        }

        // Other server error — retry with backoff
        throw new Error(`Server error ${response.status}`);
    } catch (err) {
        const newRetryCount = (op.retryCount || 0) + 1;

        if (newRetryCount >= MAX_RETRIES) {
            await updateSyncOp(op.id, { status: 'failed', retryCount: newRetryCount });
            emit('syncFailed', { op, error: err.message });
            return;
        }

        const backoffMs = BASE_BACKOFF_MS * Math.pow(2, newRetryCount);
        await updateSyncOp(op.id, { status: 'pending', retryCount: newRetryCount });
        emit('syncRetry', { op, retryCount: newRetryCount, backoffMs });

        // Schedule retry
        setTimeout(() => triggerSync(), backoffMs);
    }
}

// ── Drain the Sync Queue ──────────────────────────────────────────────────────

export async function triggerSync() {
    if (!_online || _syncing) return;

    const ops = await getPendingSyncOps();
    if (ops.length === 0) return;

    _syncing = true;
    emit('syncStart', { count: ops.length });

    for (const op of ops) {
        if (!_online) break; // Stop if we went offline mid-sync
        await processSyncOp(op);
    }

    _syncing = false;
    const remaining = await getSyncQueueCount();
    emit('syncComplete', { remaining });
}

// ── Periodic Sync (every 5s when online) ─────────────────────────────────────

setInterval(() => {
    if (_online) pullFromServer().then(() => triggerSync());
}, 5_000);

// ── Learning Content Prefetch ─────────────────────────────────────────────────
// Fetch fresh learning content from server when online and cache in IndexedDB

let _learningFetched = false;

export async function fetchAndCacheLearning() {
    if (!_online || _learningFetched) return;
    try {
        const res = await fetch(`${API_BASE}/learning`);
        if (res.ok) {
            const articles = await res.json();
            await saveLearningContent(articles);
            _learningFetched = true;
            emit('learningCached', { count: articles.length });
        }
    } catch (_) {
        // Offline or server down — use seeded defaults (handled in db.js)
    }
}

// ── Pull from Server (cross-device sync) ─────────────────────────────────────
// Fetch the latest produce & listings from the server and merge into IndexedDB.
// Records with a higher server version overwrite the local copy.
// Records that are only local (pending) are left untouched.

let _pulling = false;

export async function pullFromServer() {
    if (!_online || _pulling) return;
    _pulling = true;
    try {
        // ── Produce ──────────────────────────────────────────────────────────
        const produceRes = await fetch(`${API_BASE}/produce`);
        if (produceRes.ok) {
            const serverProduce = await produceRes.json();
            for (const item of serverProduce) {
                // Normalize server field names → client field names
                const record = {
                    id:         item.id,
                    name:       item.produceName || item.name || '',
                    category:   item.category    || '',
                    quantity:   item.quantity     || 0,
                    unit:       item.unit         || '',
                    price:      item.pricePerUnit || item.price || 0,
                    location:   item.location     || '',
                    notes:      item.notes        || '',
                    version:    item.version      || 1,
                    deleted:    item.deleted      || false,
                    createdAt:  item.createdAt    || new Date().toISOString(),
                    updatedAt:  item.updatedAt    || new Date().toISOString(),
                    syncStatus: 'synced',
                };
                await saveProduce(record);
                // Mark as synced so it doesn't get re-uploaded
                await markProduce(record.id, 'synced', record.version);
            }
        }

        // ── Listings ─────────────────────────────────────────────────────────
        const listingsRes = await fetch(`${API_BASE}/listings`);
        if (listingsRes.ok) {
            const serverListings = await listingsRes.json();
            for (const item of serverListings) {
                const record = {
                    id:          item.id,
                    produceId:   item.produceId   || '',
                    produceName: item.produceName || '',
                    quantity:    item.quantity    || 0,
                    price:       item.price       || 0,
                    location:    item.location    || '',
                    contact:     item.contact     || '',
                    status:      item.status      || 'available',
                    version:     item.version     || 1,
                    deleted:     item.deleted     || false,
                    createdAt:   item.createdAt   || new Date().toISOString(),
                    updatedAt:   item.updatedAt   || new Date().toISOString(),
                    syncStatus:  'synced',
                };
                await saveListing(record);
            }
        }

        emit('pullComplete');
    } catch (err) {
        // Offline or server error — silently continue with local data
        console.warn('Pull from server failed:', err.message);
    } finally {
        _pulling = false;
    }
}
