package services

import (
	"errors"
	"sync"

	"mavuno/internal/models"
)

var ErrNotFound = errors.New("not found")

type ProduceService struct {
	mu   sync.RWMutex
	byID map[string]models.Produce

	conflicts *ConflictService
}

func NewProduceService(conflicts *ConflictService) *ProduceService {
	svc := &ProduceService{
		byID:      make(map[string]models.Produce),
		conflicts: conflicts,
	}

	// Seed the in-memory store from SQLite on startup
	if storage.DB != nil {
		rows, err := storage.GetAllProduce()
		if err == nil {
			for _, p := range rows {
				svc.byID[p.ID] = p
			}
		}
	}

	return svc
}
