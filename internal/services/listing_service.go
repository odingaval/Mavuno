package services

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"mavuno/internal/models"
)

type ListingService struct {
	mu   sync.RWMutex
	byID map[string]models.Listing

	conflicts *ConflictService
	produce   *ProduceService
}

func NewListingService(conflicts *ConflictService, produce *ProduceService) *ListingService {
	return &ListingService{byID: make(map[string]models.Listing), conflicts: conflicts, produce: produce}
}

func (s *ListingService) List() []models.Listing {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.Listing, 0, len(s.byID))
	for _, l := range s.byID {
		out = append(out, l)
	}
	return out
}

func (s *ListingService) Get(id string) (models.Listing, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	l, ok := s.byID[id]
	return l, ok
}

func (s *ListingService) Create(l models.Listing) (models.Listing, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if l.ProduceID != "" && s.produce != nil {
		if _, ok := s.produce.Get(l.ProduceID); !ok {
			return models.Listing{}, errors.New("produce not found")
		}
	}

	if l.ID == "" {
		l.ID = fmt.Sprintf("l-%d", time.Now().UnixNano())
	}
	l.Version = 1
	l.UpdatedAt = time.Now().UnixMilli()
	s.byID[l.ID] = l
	return l, nil
}

func (s *ListingService) Patch(id string, clientVersion int, patch map[string]any) (models.Listing, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cur, ok := s.byID[id]
	if !ok {
		return models.Listing{}, ErrNotFound
	}
	if s.conflicts != nil {
		if err := s.conflicts.CheckListing(id, clientVersion, cur, true); err != nil {
			return models.Listing{}, err
		}
	}

	if v, ok := patch["produceId"].(string); ok {
		if v != "" && s.produce != nil {
			if _, ok := s.produce.Get(v); !ok {
				return models.Listing{}, errors.New("produce not found")
			}
		}
		cur.ProduceID = v
	}
	if v, ok := patch["quantity"].(float64); ok {
		cur.Quantity = v
	}
	if v, ok := patch["price"].(float64); ok {
		cur.Price = v
	}
	if v, ok := patch["location"].(string); ok {
		cur.Location = v
	}

	cur.Version++
	cur.UpdatedAt = time.Now().UnixMilli()
	s.byID[id] = cur
	return cur, nil
}

func (s *ListingService) Delete(id string, clientVersion int) (models.Listing, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cur, ok := s.byID[id]
	if !ok {
		return models.Listing{}, ErrNotFound
	}
	if s.conflicts != nil {
		if err := s.conflicts.CheckListing(id, clientVersion, cur, true); err != nil {
			return models.Listing{}, err
		}
	}
	cur.Deleted = true
	cur.Version++
	cur.UpdatedAt = time.Now().UnixMilli()
	s.byID[id] = cur
	return cur, nil
}

func (s *ListingService) UpsertFromSync(in models.Listing, clientVersion int, partial bool) (models.Listing, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cur, ok := s.byID[in.ID]
	if ok {
		if s.conflicts != nil {
			if err := s.conflicts.CheckListing(in.ID, clientVersion, cur, true); err != nil {
				return models.Listing{}, err
			}
		}
		if !partial {
			cur = in
		} else {
			if in.ProduceID != "" {
				cur.ProduceID = in.ProduceID
			}
			if in.Quantity != 0 {
				cur.Quantity = in.Quantity
			}
			if in.Price != 0 {
				cur.Price = in.Price
			}
			if in.Location != "" {
				cur.Location = in.Location
			}
			// allow Deleted explicitly
			cur.Deleted = in.Deleted
		}
		cur.Version++
		cur.UpdatedAt = time.Now().UnixMilli()
		s.byID[in.ID] = cur
		return cur, nil
	}

	if in.ProduceID != "" && s.produce != nil {
		if _, ok := s.produce.Get(in.ProduceID); !ok {
			return models.Listing{}, errors.New("produce not found")
		}
	}

	if in.ID == "" {
		in.ID = fmt.Sprintf("l-%d", time.Now().UnixNano())
	}
	in.Version = 1
	in.UpdatedAt = time.Now().UnixMilli()
	s.byID[in.ID] = in
	return in, nil
}
