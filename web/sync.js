<<<<<<< HEAD
async function syncWithServer(syncRequest) {
    try {
        const response = await fetch("/api/sync", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(syncRequest),
        });

        if (!response.ok) throw new Error("Sync failed");

        const data = await response.json();
        console.log("Sync successful:", data);

        await saveItems("produces", data.produces);
        await saveItems("listings", data.listings);

        console.log("IndexedDB updated with resolved data");
        return data;
    } catch (err) {
        console.error("Error during sync:", err);
    }
}

const testSyncRequest = {
    lastSync: new Date().toISOString(),
    produces: [
        {
            id: "p1",
            version: 1,
            updatedAt: new Date().toISOString(),
            deleted: false,
            name: "Eggs",
            quality: 120
        }
    ],
    listings: [
        {
            id: "l1",
            version: 1,
            updatedAt: new Date().toISOString(),
            deleted: false,
            produceId: "p1",
            price: 2500
        }
    ]
};

syncWithServer(testSyncRequest);
=======
/**
 * sync.js — Mavuno Background Sync Engine
 *
 * Responsibilities:
 *  - Watch online/offline transitions
 *  - Drain the syncQueue against the REST API

sync.js
7 KB
{
  "name": "Mavuno — Harvest Without Limits",
  "short_name": "Mavuno",
  "description": "Local-first PWA for smallholder farmers. Manage produce, create listings, and access learning content — fully offline.",
  "start_url": "/",
  "scope": "/",

manifest.json
3 KB
﻿
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
    triggerSync();
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

// ── Periodic Sync (every 30s when online) ────────────────────────────────────

setInterval(() => {
    if (_online) triggerSync();
}, 30_000);

// ── Learning Content Prefetch ─────────────────────────────────────────────────
// Fetch fresh learning content from server when online and cache in IndexedDB

import { saveLearningContent } from './db.js';

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
>>>>>>> ashley
