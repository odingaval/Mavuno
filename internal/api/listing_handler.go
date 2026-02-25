package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"mavuno/internal/models"
	"mavuno/internal/services"
)

var processedListingOps = struct {
	sync.RWMutex
	m map[string]bool
}{m: map[string]bool{}}

// listingHandler handles REST requests for listings using the singleton service.
type listingHandler struct {
	svc *services.ListingService
}

// Handle routes POST /api/listings, PUT /api/listings/{id}, DELETE /api/listings/{id}.
func (h *listingHandler) Handle(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	var idInPath string
	if len(parts) >= 2 {
		idInPath = parts[len(parts)-1]
		if idInPath == "listings" {
			idInPath = ""
		}
	}

	var body map[string]interface{}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
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
			jsonOK(w, map[string]string{"status": "already_processed"})
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		if idInPath != "" {
			l, ok := h.svc.Get(idInPath)
			if !ok {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			jsonOK(w, l)
		} else {
			jsonOK(w, h.svc.List())
		}
		return

	case http.MethodPost:
		l := listingFromBody("", body)
		created, err := h.svc.Create(l)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
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
		updated, err := h.svc.UpsertFromSync(listingFromBody(idInPath, body), clientVersion, false)
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
		processedListingOps.Lock()
		processedListingOps.m[opID] = true
		processedListingOps.Unlock()
	}
}

func listingFromBody(id string, body map[string]interface{}) models.Listing {
	l := models.Listing{}
	l.ID = id
	if v, ok := body["id"].(string); ok && id == "" {
		l.ID = v
	}
	if v, ok := body["produceId"].(string); ok {
		l.ProduceID = v
	}
	if v, ok := body["produceName"].(string); ok {
		l.ProduceName = v
	}
	if v, ok := body["farmerId"].(string); ok {
		l.FarmerID = v
	}
	if v, ok := body["quantity"].(float64); ok {
		l.QuantityListed = v
	}
	if v, ok := body["price"].(float64); ok {
		l.AskingPrice = v
	}
	if v, ok := body["location"].(string); ok {
		l.Location = v
	}
	if v, ok := body["contact"].(string); ok {
		l.Contact = v
	}
	if v, ok := body["status"].(string); ok {
		l.Status = models.ListingStatus(v)
	}
	if v, ok := body["version"].(float64); ok {
		l.Version = int(v)
	}
	if t, ok := body["createdAt"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			l.CreatedAt = parsed
		}
	}
	return l
}

