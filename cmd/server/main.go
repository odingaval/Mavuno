package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mavuno/internal/api"
	"mavuno/internal/services"
	"mavuno/internal/storage"
)

func main() {
	// ── Database ───────────────────────────────────────────────────────────
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./mavuno.db"
	}
	if err := storage.InitDB(dbPath); err != nil {
		log.Fatalf("Failed to initialise database: %v", err)
	}
	defer storage.CloseDB()

	if err := storage.RunMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// ── Services (singletons) ──────────────────────────────────────────────
	conflictSvc := services.NewConflictService()
	produceSvc := services.NewProduceService(conflictSvc)
	listingSvc := services.NewListingService(conflictSvc, produceSvc)
	syncSvc := services.NewSyncService(produceSvc, listingSvc, conflictSvc)

	// ── Router ─────────────────────────────────────────────────────────────
	router := api.NewRouter(produceSvc, listingSvc, syncSvc)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		log.Println("Server starting on http://localhost:8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutdown signal received, shutting down server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	log.Println("Server exited properly")
}
