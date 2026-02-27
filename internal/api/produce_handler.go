package api

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"mavuno/internal/models"
	"mavuno/internal/services"
)

type produceHandler struct {
	svc *services.ProduceService
}

func (h *produceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	var id string
	if len(parts) >= 2 && parts[len(parts)-1] != "produce" {
		id = parts[len(parts)-1]
	}

	switch r.Method {
	case http.MethodGet:
		if id != "" {
			p, ok := h.svc.Get(id)
			if !ok {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			jsonOK(w, p)
		} else {
			jsonOK(w, h.svc.List())
		}

	case http.MethodPost:
		var body map[string]interface{}
		if err := decodeBody(r, &body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		p := produceFromBody(body)
		created := h.svc.Create(p)
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

func produceFromBody(body map[string]interface{}) models.Produce {
	p := models.Produce{}
	if v, ok := body["id"].(string); ok {
		p.ID = v
	}
	if v, ok := body["name"].(string); ok {
		p.ProduceName = v
	}
	if v, ok := body["category"].(string); ok {
		p.Category = models.ProduceCategory(v)
	}
	if v, ok := body["quantity"].(float64); ok {
		p.Quantity = v
	}
	if v, ok := body["unit"].(string); ok {
		p.Unit = v
	}
	if v, ok := body["price"].(float64); ok {
		p.PricePerUnit = v
	}
	if v, ok := body["location"].(string); ok {
		p.Location = v
	}
	if v, ok := body["notes"].(string); ok {
		p.Notes = v
	}
	return p
}

func jsonOK(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

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
