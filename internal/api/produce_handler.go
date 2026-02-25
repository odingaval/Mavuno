package api

import (
	"encoding/json"
	"net/http"

	"mavuno/internal/services"

	"github.com/go-chi/chi/v5"
)

func HandleProduce(produceService *services.ProduceService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			list := produceService.List()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(list)
			return
		case http.MethodPost:
			var p services.Produce
			if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
				http.Error(w, "invalid body", http.StatusBadRequest)
				return
			}
			created := produceService.Create(p)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(created)
			return
		case http.MethodPut:
			id := chi.URLParam(r, "id")
			if id == "" {
				http.Error(w, "missing id", http.StatusBadRequest)
				return
			}
			var p services.Produce
			if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
				http.Error(w, "invalid body", http.StatusBadRequest)
				return
			}
			p.ID = id
			updated, err := produceService.Update(p)
			if err != nil {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(updated)
			return
		case http.MethodDelete:
			id := chi.URLParam(r, "id")
			if id == "" {
				http.Error(w, "missing id", http.StatusBadRequest)
				return
			}
			if err := produceService.Delete(id); err != nil {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}
}
