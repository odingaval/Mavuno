package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"mavuno/internal/models"
	"mavuno/internal/services"
)

type listingHandler struct {
	svc *services.ListingService
}

func (h *listingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	var id string
	if len(parts) >= 2 && parts[len(parts)-1] != "listings" {
		id = parts[len(parts)-1]
	}

	switch r.Method {
	case http.MethodGet:
		if id != "" {
			l, ok := h.svc.Get(id)
			if !ok {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			jsonOK(w, l)
		} else {
			jsonOK(w, h.svc.List())
		}

	case http.MethodPost:
		var body map[string]interface{}
		if err := decodeBody(r, &body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		l := listingFromBody(body)
		created, err := h.svc.Create(l)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(created)

	case http.MethodPut:
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		var body map[string]interface{}
		if err := decodeBody(r, &body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		clientVersion := 0
		if v, ok := body["version"].(float64); ok {
			clientVersion = int(v)
		}
		updated, err := h.svc.Patch(id, clientVersion, body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		jsonOK(w, updated)

	case http.MethodDelete:
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		deleted, err := h.svc.Delete(id, 0)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		jsonOK(w, deleted)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func listingFromBody(body map[string]interface{}) models.Listing {
	l := models.Listing{}
	if v, ok := body["id"].(string); ok {
		l.ID = v
	}
	if v, ok := body["produceId"].(string); ok {
		l.ProduceID = v
	}
	if v, ok := body["produceName"].(string); ok {
		l.ProduceName = v
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
	if v, ok := body["notes"].(string); ok {
		l.Notes = v
	}
	return l
}
