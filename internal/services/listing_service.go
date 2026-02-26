package services

import (
	"sync"

	"mavuno/internal/models"
)

type ListingService struct {
	mu   sync.RWMutex
	byID map[string]models.Listing

	conflicts *ConflictService
	produce   *ProduceService
}

func NewListingService(conflicts *ConflictService, produce *ProduceService) *ListingService {
	svc := &ListingService{
		byID:      make(map[string]models.Listing),
		conflicts: conflicts,
		produce:   produce,
	}

	// Seed from SQLite on startup
	if storage.DB != nil {
		rows, err := storage.GetAllListingRows()
		if err == nil {
			for _, l := range rows {
				svc.byID[l.ID] = l
			}
		}
	}

	return svc
}

func (s *ListingService) List() []models.Listing {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]models.Listing, 0, len(s.byID))

	for _, l := range s.byID {
		if !l.Deleted {
			out = append(out, l)
		}
	}

	return out
}

func (s *ListingService) Get(id string) (models.Listing, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	l, ok := s.byID[id]

	if !ok || l.Deleted {
		return models.Listing{}, false
	}

	return l, true
}
