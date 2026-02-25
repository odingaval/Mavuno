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

// syncHandler handles batched synchronization requests from the PWA.
type syncHandler struct {
	svc *services.SyncService
}

func (h *syncHandler) Handle(w http.ResponseWriter, r *http.Request) {
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

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	resp, err := h.svc.Sync(ctx, req)
	if err != nil {
		http.Error(w, "sync error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// LearningHandler serves static learning content for the PWA.
// In production this could come from a database; here we return seeded content.
func LearningHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]interface{}{}) // frontend falls back to seeded IndexedDB content
}

