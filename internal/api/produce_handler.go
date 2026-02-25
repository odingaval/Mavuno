package api

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"mavuno/internal/models"
	"mavuno/internal/services"
)

// Lightweight in-memory idempotency store for optional operation IDs.
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
	return json.NewDecoder(reader).Decode(v)
}

func jsonOK(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// produceHandler handles REST requests for produce using the singleton service.
type produceHandler struct {
	svc *services.ProduceService
}

// Handle routes POST /api/produce, PUT /api/produce/{id}, DELETE /api/produce/{id}.
func (h *produceHandler) Handle(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	var idInPath string
	if len(parts) >= 2 {
		idInPath = parts[len(parts)-1]
		// guard against trailing slash matching the base path
		if idInPath == "produce" {
			idInPath = ""
		}
	}

	var body map[string]interface{}
	if r.Body != nil {
		_ = decodeBody(r, &body)
	}

	var opID string
	if v, ok := body["operation_id"].(string); ok {
		opID = v
	}

	if opID != "" {
		processedOps.RLock()
		seen := processedOps.m[opID]
		processedOps.RUnlock()
		if seen {
			w.WriteHeader(http.StatusOK)
			jsonOK(w, map[string]string{"status": "already_processed"})
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		if idInPath != "" {
			p, ok := h.svc.Get(idInPath)
			if !ok {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			jsonOK(w, p)
		} else {
			jsonOK(w, h.svc.List())
		}
		return

	case http.MethodPost:
		p := models.Produce{}
		if name, ok := body["name"].(string); ok {
			p.ProduceName = name
		}
		if cat, ok := body["category"].(string); ok {
			p.Category = models.ProduceCategory(cat)
		}
		if qty, ok := body["quantity"].(float64); ok {
			p.Quantity = qty
		}
		if unit, ok := body["unit"].(string); ok {
			p.Unit = unit
		}
		if price, ok := body["price"].(float64); ok {
			p.PricePerUnit = price
		}
		if loc, ok := body["location"].(string); ok {
			p.Location = loc
		}
		if notes, ok := body["notes"].(string); ok {
			p.Notes = notes
		}
		if id, ok := body["id"].(string); ok {
			p.ID = id
		}
		created := h.svc.Create(p)
		w.WriteHeader(http.StatusCreated)
		jsonOK(w, created)

	case http.MethodPut:
		if idInPath == "" {
			http.Error(w, "missing id in path", http.StatusBadRequest)
			return
		}
		clientVersion := 0
		if v, ok := body["version"].(float64); ok {
			clientVersion = int(v)
		}
		updated, err := h.svc.UpsertFromSync(produceFromBody(idInPath, body), clientVersion, false)
		if err != nil {
			var ce *services.ConflictError
			if errors.As(err, &ce) {
				w.WriteHeader(http.StatusConflict)
				jsonOK(w, ce)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonOK(w, updated)

	case http.MethodDelete:
		if idInPath == "" {
			http.Error(w, "missing id in path", http.StatusBadRequest)
			return
		}
		clientVersion := 0
		if v, ok := body["version"].(float64); ok {
			clientVersion = int(v)
		}
		deleted, err := h.svc.Delete(idInPath, clientVersion)
		if err != nil {
			var ce *services.ConflictError
			if errors.As(err, &ce) {
				w.WriteHeader(http.StatusConflict)
				jsonOK(w, ce)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonOK(w, map[string]interface{}{"status": "deleted", "id": deleted.ID})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if opID != "" {
		processedOps.Lock()
		processedOps.m[opID] = true
		processedOps.Unlock()
	}
}

func produceFromBody(id string, body map[string]interface{}) models.Produce {
	p := models.Produce{}
	p.ID = id
	if name, ok := body["name"].(string); ok {
		p.ProduceName = name
	}
	if cat, ok := body["category"].(string); ok {
		p.Category = models.ProduceCategory(cat)
	}
	if qty, ok := body["quantity"].(float64); ok {
		p.Quantity = qty
	}
	if unit, ok := body["unit"].(string); ok {
		p.Unit = unit
	}
	if price, ok := body["price"].(float64); ok {
		p.PricePerUnit = price
	}
	if loc, ok := body["location"].(string); ok {
		p.Location = loc
	}
	if notes, ok := body["notes"].(string); ok {
		p.Notes = notes
	}
	if v, ok := body["version"].(float64); ok {
		p.Version = int(v)
	}
	if t, ok := body["createdAt"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			p.CreatedAt = parsed
		}
	}
	return p
}
