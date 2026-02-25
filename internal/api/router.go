package api

import (
	"net/http"

	"mavuno/internal/middleware"
	"mavuno/internal/services"

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

	// instantiate in-memory services used by handlers
	produceSvc := services.NewProduceService()
	listingSvc := services.NewListingService()

	// API routes for frontend testing
	r.Route("/api", func(api chi.Router) {
		api.Route("/produce", func(p chi.Router) {
			p.Get("/", HandleProduce(produceSvc))
			p.Post("/", HandleProduce(produceSvc))
			p.Put("/{id}", HandleProduce(produceSvc))
			p.Delete("/{id}", HandleProduce(produceSvc))
		})
		api.Route("/listings", func(l chi.Router) {
			l.Get("/", HandleListing(listingSvc))
			l.Post("/", HandleListing(listingSvc))
			l.Put("/{id}", HandleListing(listingSvc))
			l.Delete("/{id}", HandleListing(listingSvc))
		})
	})

	return r
}
