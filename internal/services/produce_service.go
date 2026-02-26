package services

import (
	"errors"
	"fmt"
	"sync"
	"time"

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

// Create creates a new produce record, persisting to SQLite.
func (s *ProduceService) Create(p models.Produce) models.Produce {
	s.mu.Lock()
	defer s.mu.Unlock()

	if p.ID == "" {
		p.ID = fmt.Sprintf("p-%d", time.Now().UnixNano())
	}

	p.Version = 1

	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	p.Deleted = false

	s.byID[p.ID] = p

	if storage.DB != nil {
		_ = storage.SaveProduce(p)
	}

	return p
}
