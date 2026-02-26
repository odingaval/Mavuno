/**
 * app.js — Mavuno PWA Application Logic
 *
 * Architecture:
 *  - Single-page app with a simple hash router
 *  - All UI updates happen optimistically (local-first)
 *  - Network is enhancement — UI never blocks waiting for it
 *  - Form state is persisted in localStorage across reloads
 */

import {
    openDB,
    generateId,
    getAllProduce,
    getProduceById,
    saveProduce,
    deleteProduce,
    getAllListings,
    saveListing,
    deleteListing,
    getAllLearning,
    getSyncQueueCount,
    enqueueSyncOp,
    seedLearningIfEmpty,
} from './db.js';

import {
    on,
    isOnline,
    triggerSync,
    fetchAndCacheLearning,
} from './sync.js';

// ── App State ─────────────────────────────────────────────────────────────────

const state = {
    currentRoute: 'dashboard',
    produce: [],
    listings: [],
    learning: [],
    pendingCount: 0,
    editingProduceId: null,
    editingListingId: null,
    notification: null,
};

// ── Routing ───────────────────────────────────────────────────────────────────

const ROUTES = {
    dashboard: renderDashboard,
    produce: renderProducePage,
    'produce-form': renderProduceForm,
    listings: renderListingsPage,
    'listing-form': renderListingForm,
    learning: renderLearningPage,
};

function navigate(route, params = {}) {
    state.currentRoute = route;
    Object.assign(state, params);
    window.location.hash = route;
    renderApp();
    window.scrollTo({ top: 0, behavior: 'smooth' });
}

window.addEventListener('hashchange', () => {
    const hash = window.location.hash.slice(1) || 'dashboard';
    const baseRoute = hash.split('?')[0];
    state.currentRoute = baseRoute;
    renderApp();
});

// ── Sync Status Badge ─────────────────────────────────────────────────────────

async function refreshPendingCount() {
    state.pendingCount = await getSyncQueueCount();
    const badge = document.getElementById('sync-badge');
    if (badge) {
        // Only show pending count when OFFLINE — when online, sync is automatic and silent
        const showBadge = !isOnline() && state.pendingCount > 0;
        badge.textContent = `${state.pendingCount} pending`;
        badge.style.display = showBadge ? 'inline-flex' : 'none';
    }
}

// ── Notification Toast ────────────────────────────────────────────────────────

function showToast(message, type = 'success') {
    const toast = document.getElementById('toast');
    if (!toast) return;
    toast.textContent = message;
    toast.className = `toast toast--${type} toast--visible`;
    clearTimeout(toast._timer);
    toast._timer = setTimeout(() => {
        toast.className = `toast toast--${type}`;
    }, 3500);
}

// ── Connectivity Status Indicator (sidebar pill + mobile dot) ─────────────────

function updateOfflineBanner() {
    const online = isOnline();

    // Sidebar status pill (desktop)
    const pill = document.getElementById('conn-status');
    const dot = document.getElementById('conn-dot');
    const label = document.getElementById('conn-label');

    if (pill) pill.classList.toggle('conn-status--offline', !online);
    if (dot) dot.title = online ? 'Online' : 'Offline';
    if (label) label.textContent = online ? 'Online' : 'Offline';

    // Mobile status dot
    const dotMobile = document.getElementById('conn-dot-mobile');
    const labelMobile = document.getElementById('conn-label-mobile');

    if (dotMobile) {
        dotMobile.style.background = online ? 'var(--color-success)' : 'var(--color-warning)';
        dotMobile.style.boxShadow = online
            ? '0 0 6px var(--color-success)'
            : '0 0 6px var(--color-warning)';
    }
    if (labelMobile) {
        labelMobile.textContent = online ? 'Online' : 'Offline';
        labelMobile.style.color = online ? 'var(--color-success)' : 'var(--color-warning)';
    }

    // Also refresh pending badge visibility (depends on online state)
    refreshPendingCount();
}

// Sync event listeners
on('online', () => {
    updateOfflineBanner();
    showToast('Back online — syncing your data…', 'success');
});
on('offline', () => {
    updateOfflineBanner();
    showToast('You are offline. Changes saved locally.', 'info');
});
on('synced', async () => {
    await refreshPendingCount();
    await loadData();
    renderApp();
});
on('syncComplete', async () => {
    await refreshPendingCount();
});
on('conflict', ({ op }) => {
    showToast(`Sync conflict on ${op.entityType} — please review.`, 'warning');
});
on('syncFailed', ({ op }) => {
    showToast(`Failed to sync ${op.entityType} after retries.`, 'error');
});

// ── Data Loading ──────────────────────────────────────────────────────────────

async function loadData() {
    [state.produce, state.listings, state.learning] = await Promise.all([
        getAllProduce(),
        getAllListings(),
        getAllLearning(),
    ]);
    state.produce.sort((a, b) => new Date(b.updatedAt) - new Date(a.updatedAt));
    state.listings.sort((a, b) => new Date(b.updatedAt) - new Date(a.updatedAt));
}

// ── Form State Persistence ────────────────────────────────────────────────────

function saveFormState(key, data) {
    try {
        localStorage.setItem(`mavuno_form_${key}`, JSON.stringify(data));
    } catch (_) { }
}

function loadFormState(key) {
    try {
        const raw = localStorage.getItem(`mavuno_form_${key}`);
        return raw ? JSON.parse(raw) : null;
    } catch (_) {
        return null;
    }
}

function clearFormState(key) {
    localStorage.removeItem(`mavuno_form_${key}`);
}

// ── Category Icons ────────────────────────────────────────────────────────────

const CATEGORY_ICONS = {
    Grain: '🌾',
    Vegetable: '🥬',
    Fruit: '🍎',
    Legume: '🫘',
    Tuber: '🥔',
    Dairy: '🥛',
    Poultry: '🐔',
    Other: '📦',
};

function categoryIcon(cat) {
    return CATEGORY_ICONS[cat] || '📦';
}

function syncStatusBadge(status) {
    if (status === 'pending' && isOnline()) return '';
    const map = {
        pending: `<span class="status-badge status-badge--pending">⏳ Pending sync</span>`,
        synced: `<span class="status-badge status-badge--synced">✅ Synced</span>`,
        conflict: `<span class="status-badge status-badge--conflict">⚠️ Conflict</span>`,
    };
    return map[status] || '';
}

// ══════════════════════════════════════════════════════════════════════════════
//  RENDER — App Shell
// ══════════════════════════════════════════════════════════════════════════════

function renderApp() {
    const main = document.getElementById('app-main');
    if (!main) return;

    const renderer = ROUTES[state.currentRoute] || renderDashboard;
    main.innerHTML = renderer();
    attachEventListeners();
    updateOfflineBanner();
    refreshPendingCount();
    updateNav();
}

function updateNav() {
    document.querySelectorAll('.nav-item').forEach((el) => {
        el.classList.toggle('nav-item--active', el.dataset.route === state.currentRoute);
    });
    document.querySelectorAll('.bottom-nav-item').forEach((el) => {
        el.classList.toggle('bottom-nav-item--active', el.dataset.route === state.currentRoute);
    });
}

// ══════════════════════════════════════════════════════════════════════════════
//  RENDER — Dashboard
// ══════════════════════════════════════════════════════════════════════════════

function renderDashboard() {
    const totalProduce = state.produce.length;
    const pendingProduce = state.produce.filter((p) => p.syncStatus === 'pending').length;
    const activeListings = state.listings.filter((l) => l.status === 'available').length;
    const totalValue = state.produce.reduce((sum, p) => sum + p.price * p.quantity, 0);

    const recentProduce = state.produce.slice(0, 3);

    return `
    <div class="page page--dashboard">
      <div class="page-header">
        <div>
          <h1 class="page-title">🌾 Dashboard</h1>
          <p class="page-subtitle">Your farm at a glance</p>
        </div>
        <div class="header-actions">
          <button class="btn btn--primary btn--sm" id="btn-add-produce-dash">+ Add Produce</button>
        </div>
      </div>

      <div class="stats-grid">
        <div class="stat-card">
          <div class="stat-card__icon">🌽</div>
          <div class="stat-card__value">${totalProduce}</div>
          <div class="stat-card__label">Total Produce Types</div>
        </div>
        <div class="stat-card">
          <div class="stat-card__icon">🛒</div>
          <div class="stat-card__value">${activeListings}</div>
          <div class="stat-card__label">Active Listings</div>
        </div>
        <div class="stat-card">
          <div class="stat-card__icon">💰</div>
          <div class="stat-card__value">KES ${totalValue.toLocaleString()}</div>
          <div class="stat-card__label">Estimated Value</div>
        </div>
        <div class="stat-card ${!isOnline() && state.pendingCount > 0 ? 'stat-card--warning' : ''}">
          <div class="stat-card__icon">${isOnline() ? (state.pendingCount > 0 ? '🔄' : '✅') : '⏳'}</div>
          <div class="stat-card__value">${isOnline() ? (state.pendingCount > 0 ? 'Syncing...' : 'Synced') : state.pendingCount}</div>
          <div class="stat-card__label">${isOnline() ? (state.pendingCount > 0 ? 'Automatic Sync' : 'All Up to Date') : 'Pending Sync'}</div>
        </div>
      </div>

      ${recentProduce.length > 0 ? `
      <div class="section">
        <div class="section-header">
          <h2 class="section-title">Recent Produce</h2>
          <button class="btn btn--ghost btn--sm" data-route-btn="produce">View All →</button>
        </div>
        <div class="card-list">
          ${recentProduce.map(produceCard).join('')}
        </div>
      </div>
      ` : `
      <div class="empty-state">
        <div class="empty-state__icon">🌱</div>
        <h2 class="empty-state__title">Start your farm inventory</h2>
        <p class="empty-state__text">Add your first produce to get started.</p>
        <button class="btn btn--primary" id="btn-add-produce-empty">+ Add Produce</button>
      </div>
      `}

      <div class="section">
        <div class="section-header">
          <h2 class="section-title">📚 Quick Learning</h2>
          <button class="btn btn--ghost btn--sm" data-route-btn="learning">View All →</button>
        </div>
        <div class="learning-quick">
          ${state.learning.slice(0, 2).map(learningCard).join('')}
        </div>
      </div>
    </div>
  `;
}

// ══════════════════════════════════════════════════════════════════════════════
//  RENDER — Produce Page
// ══════════════════════════════════════════════════════════════════════════════

function renderProducePage() {
    return `
    <div class="page">
      <div class="page-header">
        <div>
          <h1 class="page-title">🌽 My Produce</h1>
          <p class="page-subtitle">${state.produce.length} item${state.produce.length !== 1 ? 's' : ''} in inventory</p>
        </div>
        <button class="btn btn--primary" id="btn-add-produce">+ Add Produce</button>
      </div>

      ${state.produce.length === 0 ? `
        <div class="empty-state">
          <div class="empty-state__icon">🌾</div>
          <h2 class="empty-state__title">No produce yet</h2>
          <p class="empty-state__text">Add your first harvest to start managing your inventory offline.</p>
          <button class="btn btn--primary" id="btn-add-produce-empty">+ Add First Produce</button>
        </div>
      ` : `
        <div class="filter-bar">
          <input type="search" id="produce-search" class="input input--search" placeholder="🔍  Search produce…" />
        </div>
        <div class="card-list" id="produce-list">
          ${state.produce.map(produceCard).join('')}
        </div>
      `}
    </div>
  `;
}

function produceCard(p) {
    return `
    <div class="card card--produce" data-id="${p.id}">
      <div class="card__header">
        <div class="card__icon">${categoryIcon(p.category)}</div>
        <div class="card__meta">
          <h3 class="card__title">${escHtml(p.name)}</h3>
          <span class="card__sub">${escHtml(p.category)}</span>
        </div>
        ${syncStatusBadge(p.syncStatus)}
      </div>
      <div class="card__body">
        <div class="card__stat"><span>📦 Quantity</span><strong>${p.quantity} ${escHtml(p.unit)}</strong></div>
        <div class="card__stat"><span>💰 Price</span><strong>KES ${Number(p.price).toLocaleString()}/${escHtml(p.unit)}</strong></div>
        ${p.location ? `<div class="card__stat"><span>📍 Location</span><strong>${escHtml(p.location)}</strong></div>` : ''}
      </div>
      <div class="card__actions">
        <button class="btn btn--ghost btn--sm btn-edit-produce" data-id="${p.id}">✏️ Edit</button>
        <button class="btn btn--ghost btn--sm btn-list-produce" data-id="${p.id}">🛒 List</button>
        <button class="btn btn--danger btn--sm btn-delete-produce" data-id="${p.id}">🗑️</button>
      </div>
    </div>
  `;
}

// ══════════════════════════════════════════════════════════════════════════════
//  RENDER — Produce Form
// ══════════════════════════════════════════════════════════════════════════════

function renderProduceForm() {
    const isEdit = !!state.editingProduceId;
    const produce = isEdit ? state.produce.find((p) => p.id === state.editingProduceId) : null;

    // Check for persisted draft
    const draft = loadFormState('produce') || produce || {};

    const categories = Object.keys(CATEGORY_ICONS);
    const units = ['kg', 'bags', 'crates', 'tonnes', 'litres', 'pieces', 'bunches'];

    return `
    <div class="page">
      <div class="page-header">
        <button class="btn btn--ghost btn--sm btn-back" data-route="produce">← Back</button>
        <h1 class="page-title">${isEdit ? '✏️ Edit Produce' : '🌱 Add Produce'}</h1>
      </div>

      ${isEdit ? '' : '<div class="draft-notice" id="draft-notice" style="display:none">📝 Draft restored</div>'}

      <form class="form card" id="produce-form" novalidate>
        <div class="form-group">
          <label class="label" for="produce-name">Produce Name *</label>
          <input id="produce-name" name="name" type="text" class="input" placeholder="e.g. Maize, Tomatoes, Ugali" required
            value="${escHtml(draft.name || '')}" />
        </div>

        <div class="form-row">
          <div class="form-group">
            <label class="label" for="produce-category">Category *</label>
            <select id="produce-category" name="category" class="input select" required>
              <option value="">Select category…</option>
              ${categories.map(
        (c) => `<option value="${c}" ${draft.category === c ? 'selected' : ''}>${categoryIcon(c)} ${c}</option>`
    ).join('')}
            </select>
          </div>

          <div class="form-group">
            <label class="label" for="produce-unit">Unit *</label>
            <select id="produce-unit" name="unit" class="input select" required>
              ${units.map(
        (u) => `<option value="${u}" ${draft.unit === u ? 'selected' : ''}>${u}</option>`
    ).join('')}
            </select>
          </div>
        </div>

        <div class="form-row">
          <div class="form-group">
            <label class="label" for="produce-quantity">Quantity *</label>
            <input id="produce-quantity" name="quantity" type="number" class="input" placeholder="0" min="0" step="0.1" required
              value="${draft.quantity || ''}" />
          </div>

          <div class="form-group">
            <label class="label" for="produce-price">Price per unit (KES) *</label>
            <input id="produce-price" name="price" type="number" class="input" placeholder="0.00" min="0" step="0.01" required
              value="${draft.price || ''}" />
          </div>
        </div>

        <div class="form-group">
          <label class="label" for="produce-location">Location / Farm</label>
          <input id="produce-location" name="location" type="text" class="input" placeholder="e.g. Nakuru North, Plot 12"
            value="${escHtml(draft.location || '')}" />
        </div>

        <div class="form-group">
          <label class="label" for="produce-notes">Additional Notes</label>
          <textarea id="produce-notes" name="notes" class="input textarea" rows="3" placeholder="Quality notes, harvest date, etc.">${escHtml(draft.notes || '')}</textarea>
        </div>

        <div class="form-actions">
          <button type="button" class="btn btn--ghost" data-route="produce">Cancel</button>
          <button type="submit" class="btn btn--primary" id="submit-produce">
            ${isEdit ? '💾 Save Changes' : '✅ Add Produce'}
          </button>
        </div>
      </form>
    </div>
  `;
}

// ══════════════════════════════════════════════════════════════════════════════
//  RENDER — Listings Page
// ══════════════════════════════════════════════════════════════════════════════

function renderListingsPage() {
    const available = state.listings.filter((l) => l.status === 'available');
    const other = state.listings.filter((l) => l.status !== 'available');

    return `
    <div class="page">
      <div class="page-header">
        <div>
          <h1 class="page-title">🛒 Market Listings</h1>
          <p class="page-subtitle">${available.length} active listing${available.length !== 1 ? 's' : ''}</p>
        </div>
        <button class="btn btn--primary" id="btn-add-listing">+ New Listing</button>
      </div>

      ${state.listings.length === 0 ? `
        <div class="empty-state">
          <div class="empty-state__icon">🛒</div>
          <h2 class="empty-state__title">No listings yet</h2>
          <p class="empty-state__text">Create a market listing to connect with buyers — even offline.</p>
          <button class="btn btn--primary" id="btn-add-listing-empty">+ Create Listing</button>
        </div>
      ` : `
        ${available.length > 0 ? `
          <div class="section">
            <h2 class="section-title">Active Listings</h2>
            <div class="card-list">${available.map(listingCard).join('')}</div>
          </div>
        ` : ''}
        ${other.length > 0 ? `
          <div class="section">
            <h2 class="section-title">Closed Listings</h2>
            <div class="card-list card-list--muted">${other.map(listingCard).join('')}</div>
          </div>
        ` : ''}
      `}
    </div>
  `;
}

function listingCard(l) {
    const statusColors = { available: 'green', sold: 'blue', expired: 'gray' };
    const color = statusColors[l.status] || 'gray';
    return `
    <div class="card card--listing" data-id="${l.id}">
      <div class="card__header">
        <div class="card__icon">🛒</div>
        <div class="card__meta">
          <h3 class="card__title">${escHtml(l.produceName || 'Unnamed Produce')}</h3>
          <span class="pill pill--${color}">${l.status}</span>
        </div>
        ${syncStatusBadge(l.syncStatus)}
      </div>
      <div class="card__body">
        <div class="card__stat"><span>📦 Qty</span><strong>${l.quantity}</strong></div>
        <div class="card__stat"><span>💰 Price</span><strong>KES ${Number(l.price).toLocaleString()}</strong></div>
        ${l.location ? `<div class="card__stat"><span>📍</span><strong>${escHtml(l.location)}</strong></div>` : ''}
        ${l.contact ? `<div class="card__stat"><span>📞</span><strong>${escHtml(l.contact)}</strong></div>` : ''}
      </div>
      <div class="card__actions">
        <button class="btn btn--ghost btn--sm btn-edit-listing" data-id="${l.id}">✏️ Edit</button>
        <button class="btn btn--danger btn--sm btn-delete-listing" data-id="${l.id}">🗑️</button>
      </div>
    </div>
  `;
}

// ══════════════════════════════════════════════════════════════════════════════
//  RENDER — Listing Form
// ══════════════════════════════════════════════════════════════════════════════

function renderListingForm() {
    const isEdit = !!state.editingListingId;
    const listing = isEdit ? state.listings.find((l) => l.id === state.editingListingId) : null;
    const draft = loadFormState('listing') || listing || {};

    // Produce selection options
    const produceOptions = state.produce.map(
        (p) => `<option value="${p.id}" ${draft.produceId === p.id ? 'selected' : ''}>${categoryIcon(p.category)} ${escHtml(p.name)}</option>`
    ).join('');

    const statuses = ['available', 'sold', 'expired'];

    return `
    <div class="page">
      <div class="page-header">
        <button class="btn btn--ghost btn--sm btn-back" data-route="listings">← Back</button>
        <h1 class="page-title">${isEdit ? '✏️ Edit Listing' : '🛒 New Listing'}</h1>
      </div>

      <form class="form card" id="listing-form" novalidate>
        <div class="form-group">
          <label class="label" for="listing-produce">Produce *</label>
          ${state.produce.length > 0 ? `
            <select id="listing-produce" name="produceId" class="input select" required>
              <option value="">Select produce…</option>
              ${produceOptions}
            </select>
          ` : `
            <div class="info-box">
              You need to add produce first before creating a listing.
              <button type="button" class="btn btn--sm btn--primary" data-route-btn="produce-form">+ Add Produce</button>
            </div>
          `}
        </div>

        <div class="form-row">
          <div class="form-group">
            <label class="label" for="listing-quantity">Quantity</label>
            <input id="listing-quantity" name="quantity" type="number" class="input" placeholder="0" min="0" step="0.1"
              value="${draft.quantity || ''}" />
          </div>

          <div class="form-group">
            <label class="label" for="listing-price">Asking Price (KES)</label>
            <input id="listing-price" name="price" type="number" class="input" placeholder="0.00" min="0" step="0.01"
              value="${draft.price || ''}" />
          </div>
        </div>

        <div class="form-group">
          <label class="label" for="listing-location">Pickup Location</label>
          <input id="listing-location" name="location" type="text" class="input" placeholder="e.g. Eldoret Market, Gate 3"
            value="${escHtml(draft.location || '')}" />
        </div>

        <div class="form-group">
          <label class="label" for="listing-contact">Contact Number</label>
          <input id="listing-contact" name="contact" type="tel" class="input" placeholder="e.g. 0712 345 678"
            value="${escHtml(draft.contact || '')}" />
        </div>

        ${isEdit ? `
          <div class="form-group">
            <label class="label" for="listing-status">Status</label>
            <select id="listing-status" name="status" class="input select">
              ${statuses.map((s) => `<option value="${s}" ${(draft.status || 'available') === s ? 'selected' : ''}>${s}</option>`).join('')}
            </select>
          </div>
        ` : ''}

        <div class="form-actions">
          <button type="button" class="btn btn--ghost" data-route="listings">Cancel</button>
          <button type="submit" class="btn btn--primary">
            ${isEdit ? '💾 Save Changes' : '✅ Create Listing'}
          </button>
        </div>
      </form>
    </div>
  `;
}

// ══════════════════════════════════════════════════════════════════════════════
//  RENDER — Learning Page
// ══════════════════════════════════════════════════════════════════════════════

function renderLearningPage() {
    const categories = [...new Set(state.learning.map((a) => a.category))];

    return `
    <div class="page">
      <div class="page-header">
        <div>
          <h1 class="page-title">📚 Learning Centre</h1>
          <p class="page-subtitle">Agricultural knowledge — available offline</p>
        </div>
      </div>

      ${state.learning.length === 0 ? `
        <div class="empty-state">
          <div class="empty-state__icon">📖</div>
          <h2 class="empty-state__title">No content yet</h2>
          <p class="empty-state__text">Connect to the internet once to download learning materials.</p>
        </div>
      ` : `
        <div class="filter-bar">
          <input type="search" id="learning-search" class="input input--search" placeholder="🔍  Search articles…" />
        </div>
        ${categories.map((cat) => `
          <div class="section">
            <h2 class="section-title">${cat}</h2>
            <div class="learning-grid">
              ${state.learning.filter((a) => a.category === cat).map(learningCard).join('')}
            </div>
          </div>
        `).join('')}
      `}
    </div>
  `;
}

function learningCard(article) {
    return `
    <div class="card card--learning" data-id="${article.id}">
      <div class="card__header">
        <span class="pill pill--purple">${escHtml(article.category)}</span>
      </div>
      <h3 class="card__title">${escHtml(article.title)}</h3>
      <div class="article-body">${escHtml(article.body).replace(/\n/g, '<br>').replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')}</div>
      <div class="card__tags">
        ${(article.tags || []).map((t) => `<span class="tag">#${escHtml(t)}</span>`).join('')}
      </div>
    </div>
  `;
}

// ══════════════════════════════════════════════════════════════════════════════
//  EVENT LISTENERS
// ══════════════════════════════════════════════════════════════════════════════

function attachEventListeners() {
    // ── Navigation buttons ────────────────────────────────────────────────────
    document.querySelectorAll('[data-route-btn]').forEach((btn) => {
        btn.addEventListener('click', () => navigate(btn.dataset.routeBtn));
    });

    document.querySelectorAll('[data-route]').forEach((el) => {
        el.addEventListener('click', () => {
            if (el.dataset.route) {
                state.editingProduceId = null;
                state.editingListingId = null;
                navigate(el.dataset.route);
            }
        });
    });

    // ── Produce Page ──────────────────────────────────────────────────────────
    document.getElementById('btn-add-produce')?.addEventListener('click', () => {
        state.editingProduceId = null;
        navigate('produce-form');
    });
    document.getElementById('btn-add-produce-dash')?.addEventListener('click', () => {
        state.editingProduceId = null;
        navigate('produce-form');
    });
    document.getElementById('btn-add-produce-empty')?.addEventListener('click', () => {
        state.editingProduceId = null;
        navigate('produce-form');
    });

    document.querySelectorAll('.btn-edit-produce').forEach((btn) => {
        btn.addEventListener('click', () => {
            state.editingProduceId = btn.dataset.id;
            clearFormState('produce');
            navigate('produce-form');
        });
    });

    document.querySelectorAll('.btn-list-produce').forEach((btn) => {
        btn.addEventListener('click', async () => {
            const produce = state.produce.find((p) => p.id === btn.dataset.id);
            if (produce) {
                saveFormState('listing', {
                    produceId: produce.id,
                    quantity: produce.quantity,
                    price: produce.price,
                    location: produce.location || '',
                });
                state.editingListingId = null;
                navigate('listing-form');
            }
        });
    });

    document.querySelectorAll('.btn-delete-produce').forEach((btn) => {
        btn.addEventListener('click', async () => {
            if (!confirm('Delete this produce? This cannot be undone.')) return;
            const id = btn.dataset.id;

            // Optimistic UI: remove immediately
            state.produce = state.produce.filter((p) => p.id !== id);
            renderApp();

            await deleteProduce(id);
            await enqueueSyncOp({ entityType: 'produce', operation: 'delete', entityId: id, payload: {} });
            await refreshPendingCount();
            showToast('Produce deleted. Will sync when online.', 'info');
            triggerSync();
        });
    });

    // ── Produce Form ──────────────────────────────────────────────────────────
    const produceForm = document.getElementById('produce-form');
    if (produceForm) {
        // Auto-save draft on input
        produceForm.addEventListener('input', () => {
            const data = Object.fromEntries(new FormData(produceForm));
            saveFormState('produce', data);
        });

        // Check for existing draft
        const draft = loadFormState('produce');
        if (draft && !state.editingProduceId) {
            const notice = document.getElementById('draft-notice');
            if (notice) notice.style.display = 'block';
        }

        produceForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            if (!produceForm.checkValidity()) {
                produceForm.reportValidity();
                return;
            }

            const formData = Object.fromEntries(new FormData(produceForm));
            const btn = document.getElementById('submit-produce');
            if (btn) { btn.disabled = true; btn.textContent = 'Saving…'; }

            const isEdit = !!state.editingProduceId;
            const produce = {
                id: isEdit ? state.editingProduceId : generateId(),
                name: formData.name.trim(),
                category: formData.category,
                unit: formData.unit,
                quantity: parseFloat(formData.quantity) || 0,
                price: parseFloat(formData.price) || 0,
                location: formData.location?.trim() || '',
                notes: formData.notes?.trim() || '',
                version: isEdit ? (state.produce.find((p) => p.id === state.editingProduceId)?.version || 1) : 1,
                syncStatus: 'pending',
                createdAt: isEdit
                    ? (state.produce.find((p) => p.id === state.editingProduceId)?.createdAt || new Date().toISOString())
                    : new Date().toISOString(),
            };

            // Optimistic update
            if (isEdit) {
                state.produce = state.produce.map((p) => (p.id === produce.id ? { ...p, ...produce } : p));
            } else {
                state.produce.unshift(produce);
            }

            clearFormState('produce');
            state.editingProduceId = null;
            navigate('produce');
            showToast(isEdit ? 'Produce updated. Syncing…' : 'Produce added. Syncing…', 'success');

            // Persist locally
            await saveProduce(produce);

            // Enqueue sync
            await enqueueSyncOp({
                entityType: 'produce',
                operation: isEdit ? 'update' : 'create',
                entityId: produce.id,
                payload: produce,
            });

            await refreshPendingCount();
            triggerSync();
        });
    }

    // ── Listings Page ─────────────────────────────────────────────────────────
    document.getElementById('btn-add-listing')?.addEventListener('click', () => {
        state.editingListingId = null;
        navigate('listing-form');
    });
    document.getElementById('btn-add-listing-empty')?.addEventListener('click', () => {
        state.editingListingId = null;
        navigate('listing-form');
    });

    document.querySelectorAll('.btn-edit-listing').forEach((btn) => {
        btn.addEventListener('click', () => {
            state.editingListingId = btn.dataset.id;
            clearFormState('listing');
            navigate('listing-form');
        });
    });

    document.querySelectorAll('.btn-delete-listing').forEach((btn) => {
        btn.addEventListener('click', async () => {
            if (!confirm('Delete this listing?')) return;
            const id = btn.dataset.id;

            state.listings = state.listings.filter((l) => l.id !== id);
            renderApp();

            await deleteListing(id);
            await enqueueSyncOp({ entityType: 'listing', operation: 'delete', entityId: id, payload: {} });
            await refreshPendingCount();
            showToast('Listing deleted.', 'info');
            triggerSync();
        });
    });

    // ── Listing Form ──────────────────────────────────────────────────────────
    const listingForm = document.getElementById('listing-form');
    if (listingForm) {
        listingForm.addEventListener('input', () => {
            const data = Object.fromEntries(new FormData(listingForm));
            saveFormState('listing', data);
        });

        listingForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            if (!listingForm.checkValidity()) {
                listingForm.reportValidity();
                return;
            }

            const formData = Object.fromEntries(new FormData(listingForm));
            const isEdit = !!state.editingListingId;

            const selectedProduce = state.produce.find((p) => p.id === formData.produceId);

            const listing = {
                id: isEdit ? state.editingListingId : generateId(),
                produceId: formData.produceId,
                produceName: selectedProduce?.name || '',
                quantity: parseFloat(formData.quantity) || 0,
                price: parseFloat(formData.price) || 0,
                location: formData.location?.trim() || '',
                contact: formData.contact?.trim() || '',
                status: formData.status || 'available',
                version: isEdit ? (state.listings.find((l) => l.id === state.editingListingId)?.version || 1) : 1,
                syncStatus: 'pending',
                createdAt: isEdit
                    ? (state.listings.find((l) => l.id === state.editingListingId)?.createdAt || new Date().toISOString())
                    : new Date().toISOString(),
            };

            // Optimistic update
            if (isEdit) {
                state.listings = state.listings.map((l) => (l.id === listing.id ? { ...l, ...listing } : l));
            } else {
                state.listings.unshift(listing);
            }

            clearFormState('listing');
            state.editingListingId = null;
            navigate('listings');
            showToast(isEdit ? 'Listing updated. Syncing…' : 'Listing created. Syncing…', 'success');

            await saveListing(listing);
            await enqueueSyncOp({
                entityType: 'listing',
                operation: isEdit ? 'update' : 'create',
                entityId: listing.id,
                payload: listing,
            });
            await refreshPendingCount();
            triggerSync();
        });
    }

    // ── Search filtering ──────────────────────────────────────────────────────
    document.getElementById('produce-search')?.addEventListener('input', (e) => {
        const q = e.target.value.toLowerCase();
        document.querySelectorAll('#produce-list .card--produce').forEach((card) => {
            const text = card.textContent.toLowerCase();
            card.style.display = text.includes(q) ? '' : 'none';
        });
    });

    document.getElementById('learning-search')?.addEventListener('input', (e) => {
        const q = e.target.value.toLowerCase();
        document.querySelectorAll('.card--learning').forEach((card) => {
            const text = card.textContent.toLowerCase();
            card.style.display = text.includes(q) ? '' : 'none';
        });
    });
}

// ── Nav event listeners (persistent shell) ────────────────────────────────────

function attachShellListeners() {
    // Sidebar nav (desktop)
    document.querySelectorAll('.nav-item').forEach((el) => {
        el.addEventListener('click', () => {
            state.editingProduceId = null;
            state.editingListingId = null;
            navigate(el.dataset.route);
        });
        // Keyboard accessibility
        el.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                el.click();
            }
        });
    });

    // Bottom nav (mobile)
    document.querySelectorAll('.bottom-nav-item').forEach((el) => {
        el.addEventListener('click', () => {
            state.editingProduceId = null;
            state.editingListingId = null;
            navigate(el.dataset.route);
            // Update active state on bottom nav
            document.querySelectorAll('.bottom-nav-item').forEach((item) => {
                item.classList.toggle('bottom-nav-item--active', item.dataset.route === el.dataset.route);
            });
        });
    });

    // Sync button
    document.getElementById('btn-sync')?.addEventListener('click', () => {
        if (isOnline()) {
            triggerSync();
            showToast('Syncing…', 'success');
        } else {
            showToast('You are offline. Sync will run when online.', 'info');
        }
    });
}

// ── Escape HTML helper ────────────────────────────────────────────────────────

function escHtml(str) {
    if (!str) return '';
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

// ══════════════════════════════════════════════════════════════════════════════
//  SERVICE WORKER REGISTRATION
// ══════════════════════════════════════════════════════════════════════════════

async function registerSW() {
    if (!('serviceWorker' in navigator)) return;
    try {
        const reg = await navigator.serviceWorker.register('/sw.js', { scope: '/' });
        reg.addEventListener('updatefound', () => {
            // New SW available — show refresh prompt
            const worker = reg.installing;
            worker.addEventListener('statechange', () => {
                if (worker.state === 'installed' && navigator.serviceWorker.controller) {
                    showToast('App updated! Refresh for the latest version.', 'info');
                }
            });
        });
    } catch (err) {
        console.warn('SW registration failed:', err);
    }
}

// ══════════════════════════════════════════════════════════════════════════════
//  BOOTSTRAP
// ══════════════════════════════════════════════════════════════════════════════

async function init() {
    await openDB();
    await seedLearningIfEmpty();
    await loadData();

    // Restore route from URL hash
    const hash = window.location.hash.slice(1);
    if (hash && ROUTES[hash]) state.currentRoute = hash;

    renderApp();
    attachShellListeners();
    updateOfflineBanner();

    // Register Service Worker
    await registerSW();

    // Listen for background sync messages from the Service Worker
    if ('serviceWorker' in navigator) {
        navigator.serviceWorker.addEventListener('message', (event) => {
            if (event.data?.type === 'BACKGROUND_SYNC') {
                triggerSync();
            }
        });
    }

    // Background tasks
    fetchAndCacheLearning();
    triggerSync();
}

init().catch(console.error);
