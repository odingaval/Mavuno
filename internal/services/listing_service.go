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
	svc := &ListingService{byID: make(map[string]models.Listing), conflicts: conflicts, produce: produce}

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
