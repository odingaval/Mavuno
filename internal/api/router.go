package api

import (
	"net/http"

	"mavuno/internal/middleware"

	"github.com/go-chi/chi/v5"
)

// NewRouter sets up routes, middleware and CORS for the server
func NewRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Mavuno API is running 🚜"))
	})
	r.Get("/fail", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Simulated server error", http.StatusInternalServerError)
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// API routes for frontend testing
	r.Route("/api", func(api chi.Router) {
		api.Route("/produce", func(p chi.Router) {
			p.Post("/", ProduceHandler)
			p.Put("/{id}", ProduceHandler)
			p.Delete("/{id}", ProduceHandler)
		})
		api.Route("/listings", func(l chi.Router) {
			l.Post("/", ListingHandler)
			l.Put("/{id}", ListingHandler)
			l.Delete("/{id}", ListingHandler)
		})
	})

	return r
}
