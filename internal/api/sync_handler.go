package api

import (
	"encoding/json"
	"net/http"

	"mavuno/internal/services"
)

func HandleSync(syncService *services.SyncService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req services.SyncRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		resp := syncService.Sync(req)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}
