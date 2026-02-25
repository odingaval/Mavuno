package api

import (
	"net/http"

	"mavuno/internal/services"
)

func SetupRouter() {
	conflictService := services.NewConflictService()
	syncService := services.NewSyncService(conflictService)

	http.HandleFunc("/api/sync", HandleSync(syncService))
}
