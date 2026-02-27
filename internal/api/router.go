package api

import (
	"net/http"

	"mavuno/internal/middleware"
	"mavuno/internal/services"
)

// NewRouter wires the HTTP routes for the backend.
func NewRouter(produceSvc *services.ProduceService, listingSvc *services.ListingService, syncSvc *services.SyncService) http.Handler {
	mux := http.NewServeMux()

	// ── API routes (prefixed with /api/) ──────────────────────────────────
	mux.Handle("/api/produce", &produceHandler{svc: produceSvc})
	mux.Handle("/api/produce/", &produceHandler{svc: produceSvc})
	mux.Handle("/api/listings", &listingHandler{svc: listingSvc})
	mux.Handle("/api/listings/", &listingHandler{svc: listingSvc})
	mux.Handle("/api/sync", &syncHandler{svc: syncSvc})
	mux.HandleFunc("/api/learning", LearningHandler)

	// ── Static file server — serves the web/ PWA ─────────────────────────
	// Serves index.html, app.js, db.js, sync.js, styles.css, sw.js, etc.
	fs := http.FileServer(http.Dir("./web"))
	mux.Handle("/", fs)

	return middleware.Logging(mux)
}
