package api

import (
	"net/http"

	"mavuno/internal/middleware"
)

// NewRouter wires the HTTP routes for the backend.
func NewRouter() http.Handler {
	mux := http.NewServeMux()

	// existing basic endpoints (non-sync)
	mux.HandleFunc("/produce", ProduceHandler)
	mux.HandleFunc("/produce/", ProduceHandler)
	mux.HandleFunc("/listings", ListingHandler)
	mux.HandleFunc("/listings/", ListingHandler)

	mux.HandleFunc("/sync", SyncHandler)

	return middleware.Logging(mux)
}
