package api

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Lightweight in-memory idempotency store for optional operation IDs.
// This is fine for local testing. A production server should persist this.
var processedOps = struct {
	sync.RWMutex
	m map[string]bool
}{m: map[string]bool{}}

// decodeBody decodes JSON request bodies and supports optional gzip encoding.
func decodeBody(r *http.Request, v interface{}) error {
	var reader io.Reader = r.Body
	if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			return err
		}
		defer gz.Close()
		reader = gz
	}
	decoder := json.NewDecoder(reader)
	return decoder.Decode(v)
}

// ProduceHandler accepts simple REST-style requests from the frontend:
//   - POST /produce         -> create (body: produce JSON)
//   - PUT  /produce/{id}    -> update (body: produce JSON)
//   - DELETE /produce/{id}  -> delete
//
// It also understands an optional `operation_id` field for idempotency.
func ProduceHandler(w http.ResponseWriter, r *http.Request) {
	// Determine ID from URL if present.
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	var idInPath string
	if len(parts) >= 2 {
		idInPath = parts[len(parts)-1]
	}

	// Read body if present
	var body map[string]interface{}
	if r.Body != nil {
		_ = decodeBody(r, &body) // ignore decode errors for empty bodies
	}

	// optional operation id
	var opID string
	if v, ok := body["operation_id"].(string); ok {
		opID = v
	}

	// idempotency check if opID provided
	if opID != "" {
		processedOps.RLock()
		seen := processedOps.m[opID]
		processedOps.RUnlock()
		if seen {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "already_processed"})
			return
		}
	}

	// Basic behaviour: accept the request and respond 200/201 so the frontend
	// can consider the op processed. This is intentionally simple to allow
	// testing without a full backend implementation.
	switch r.Method {
	case http.MethodPost:
		// create
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "created", "created_at": time.Now()})
	case http.MethodPut:
		// update
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

	// mark operation processed if provided
	if opID != "" {
		processedOps.Lock()
		processedOps.m[opID] = true
		processedOps.Unlock()
	}
}
