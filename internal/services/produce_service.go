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

func (s *ProduceService) List() []models.Produce {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]models.Produce, 0, len(s.byID))

	for _, p := range s.byID {
		if !p.Deleted {
			out = append(out, p)
		}
	}

	return out
}

func (s *ProduceService) Get(id string) (models.Produce, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.byID[id]

	if !ok || p.Deleted {
		return models.Produce{}, false
	}

	return p, true
}
