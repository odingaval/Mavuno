package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

var listingStore = struct {
	sync.RWMutex
	m map[string]map[string]interface{}
}{m: make(map[string]map[string]interface{})}

var processedListingOps = struct {
	sync.RWMutex
	m map[string]bool
}{m: make(map[string]bool)}

// ListingHandler handles create/update/delete for listing resources.
func ListingHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	var idInPath string
	if len(parts) >= 3 {
		idInPath = parts[len(parts)-1]
	}

	var body map[string]interface{}
	if r.Body != nil {
		_ = decodeJSONBody(r, &body)
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
		id := ""
		if v, ok := body["id"].(string); ok {
			id = v
		}
		if id == "" {
			id = time.Now().Format("20060102150405.000000")
		}
		listingStore.Lock()
		listingStore.m[id] = body
		listingStore.Unlock()
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "created", "id": id})
	case http.MethodPut:
		if idInPath == "" {
			http.Error(w, "missing id in path", http.StatusBadRequest)
			return
		}
		listingStore.Lock()
		listingStore.m[idInPath] = body
		listingStore.Unlock()
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "updated", "id": idInPath})
	case http.MethodDelete:
		if idInPath == "" {
			http.Error(w, "missing id in path", http.StatusBadRequest)
			return
		}
		listingStore.Lock()
		delete(listingStore.m, idInPath)
		listingStore.Unlock()
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
