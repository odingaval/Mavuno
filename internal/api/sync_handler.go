package api

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"mavuno/internal/services"
)

// SyncHandler handles batched synchronization requests from the PWA.
// It contains no business logic; it delegates to SyncService.
func SyncHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reader io.Reader = r.Body
	if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, "invalid gzip", http.StatusBadRequest)
			return
		}
		defer gz.Close()
		reader = gz
	}

	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()

	var req services.SyncRequest
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Instantiate services (in-memory). This can later be wired with a persistent repository.
	conflicts := services.NewConflictService()
	produceSvc := services.NewProduceService(conflicts)
	listingSvc := services.NewListingService(conflicts, produceSvc)
	syncSvc := services.NewSyncService(produceSvc, listingSvc, conflicts)

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	resp, err := syncSvc.Sync(ctx, req)
	if err != nil {
		http.Error(w, "sync error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
