package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

var produceStore = struct {
	sync.RWMutex
	m map[string]map[string]interface{}
}{m: make(map[string]map[string]interface{})}

var processedProduceOps = struct {
	sync.RWMutex
	m map[string]bool
}{m: make(map[string]bool)}

// ProduceHandler handles create/update/delete for produce resources.
func ProduceHandler(w http.ResponseWriter, r *http.Request) {
	// path may be /api/produce or /api/produce/{id}
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

	// optional operation id for idempotency
	var opID string
	if v, ok := body["operation_id"].(string); ok {
		opID = v
	}

	if opID != "" {
		processedProduceOps.RLock()
		seen := processedProduceOps.m[opID]
		processedProduceOps.RUnlock()
		if seen {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "already_processed"})
			return
		}
	}

	switch r.Method {
	case http.MethodPost:
		// create - expect body contains id and data
		id := ""
		if v, ok := body["id"].(string); ok {
			id = v
		}
		if id == "" {
			id = time.Now().Format("20060102150405.000000")
		}
		produceStore.Lock()
		produceStore.m[id] = body
		produceStore.Unlock()
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "created", "id": id})
	case http.MethodPut:
		if idInPath == "" {
			http.Error(w, "missing id in path", http.StatusBadRequest)
			return
		}
		produceStore.Lock()
		produceStore.m[idInPath] = body
		produceStore.Unlock()
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "updated", "id": idInPath})
	case http.MethodDelete:
		if idInPath == "" {
			http.Error(w, "missing id in path", http.StatusBadRequest)
			return
		}
		produceStore.Lock()
		delete(produceStore.m, idInPath)
		produceStore.Unlock()
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "deleted", "id": idInPath})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if opID != "" {
		processedProduceOps.Lock()
		processedProduceOps.m[opID] = true
		processedProduceOps.Unlock()
	}
}
