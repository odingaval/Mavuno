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
	ph := &produceHandler{svc: produceSvc}
	mux.HandleFunc("/api/produce", ph.Handle)
	mux.HandleFunc("/api/produce/", ph.Handle)

	lh := &listingHandler{svc: listingSvc}
	mux.HandleFunc("/api/listings", lh.Handle)
	mux.HandleFunc("/api/listings/", lh.Handle)

	sh := &syncHandler{svc: syncSvc}
	mux.HandleFunc("/api/sync", sh.Handle)

	// ── Learning content endpoint ─────────────────────────────────────────
	mux.HandleFunc("/api/learning", LearningHandler)

	// ── Static file server — serves the web/ PWA ─────────────────────────
	fs := http.FileServer(http.Dir("./web"))
	mux.Handle("/", fs)

	return middleware.Logging(mux)
}
