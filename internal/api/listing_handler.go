package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Reuse the in-memory idempotency store used by produce handler.
var processedListingOps = struct {
	sync.RWMutex
	m map[string]bool
}{m: map[string]bool{}}

func decodeListingBody(r *http.Request, v interface{}) error {
	// simple JSON decode; listing payloads are not gzipped in this flow
	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(v)
}

// ListingHandler handles POST/PUT/DELETE for listings from the frontend.
func ListingHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	var idInPath string
	if len(parts) >= 2 {
		idInPath = parts[len(parts)-1]
	}

	var body map[string]interface{}
	if r.Body != nil {
		_ = decodeListingBody(r, &body)
	}

	var opID string
	if v, ok := body["operation_id"].(string); ok {
		opID = v
	}

	if opID != "" {
		processedListingOps.RLock()
		seen := processedListingOps.m[opID]
		processedListingOps.RUnlock()
		if seen {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "already_processed"})
			return
		}
	}

	switch r.Method {
	case http.MethodPost:
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "created", "created_at": time.Now()})
	case http.MethodPut:
		if idInPath == "" {
			http.Error(w, "missing id in path", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "updated", "id": idInPath, "updated_at": time.Now()})
	case http.MethodDelete:
		if idInPath == "" {
			http.Error(w, "missing id in path", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "deleted", "id": idInPath})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if opID != "" {
		processedListingOps.Lock()
		processedListingOps.m[opID] = true
		processedListingOps.Unlock()
	}
}
