package services

import (
	"fmt"
	"sync"
	"time"

	"mavuno/internal/models"
	"mavuno/internal/storage"
)

type ListingService struct {
	mu   sync.RWMutex
	byID map[string]models.Listing

	conflicts *ConflictService
	produce   *ProduceService
}

func NewListingService(conflicts *ConflictService, produce *ProduceService) *ListingService {
	svc := &ListingService{byID: make(map[string]models.Listing), conflicts: conflicts, produce: produce}
	if storage.DB != nil {
		if rows, err := storage.GetAllListingRows(); err == nil {
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

func (s *ListingService) Create(l models.Listing) (models.Listing, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if l.ID == "" {
		l.ID = fmt.Sprintf("l-%d", time.Now().UnixNano())
	}
	l.Version = 1
	now := time.Now()
	l.CreatedAt = now
	l.UpdatedAt = now
	l.Deleted = false
	if l.Status == "" {
		l.Status = models.StatusAvailable
	}
	s.byID[l.ID] = l
	if storage.DB != nil {
		_ = storage.SaveListing(l)
	}
	return l, nil
}

func (s *ListingService) Patch(id string, clientVersion int, patch map[string]any) (models.Listing, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cur, ok := s.byID[id]
	if !ok || cur.Deleted {
		return models.Listing{}, ErrNotFound
	}
	if s.conflicts != nil {
		if err := s.conflicts.CheckListing(id, clientVersion, cur, true); err != nil {
			return models.Listing{}, err
		}
	}
	if v, ok := patch["produceId"].(string); ok {
		cur.ProduceID = v
	}
	if v, ok := patch["produceName"].(string); ok {
		cur.ProduceName = v
	}
	if v, ok := patch["quantity"].(float64); ok {
		cur.QuantityListed = v
	}
	if v, ok := patch["price"].(float64); ok {
		cur.AskingPrice = v
	}
	if v, ok := patch["location"].(string); ok {
		cur.Location = v
	}
	if v, ok := patch["contact"].(string); ok {
		cur.Contact = v
	}
	if v, ok := patch["status"].(string); ok {
		cur.Status = models.ListingStatus(v)
	}
	cur.Version++
	cur.UpdatedAt = time.Now()
	s.byID[id] = cur
	if storage.DB != nil {
		_ = storage.SaveListing(cur)
	}
	return cur, nil
}

func (s *ListingService) Delete(id string, clientVersion int) (models.Listing, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cur, ok := s.byID[id]
	if !ok || cur.Deleted {
		return models.Listing{}, ErrNotFound
	}
	if s.conflicts != nil {
		if err := s.conflicts.CheckListing(id, clientVersion, cur, true); err != nil {
			return models.Listing{}, err
		}
	}
	cur.Deleted = true
	cur.Version++
	cur.UpdatedAt = time.Now()
	s.byID[id] = cur
	if storage.DB != nil {
		_ = storage.SaveListing(cur)
	}
	return cur, nil
}

func (s *ListingService) UpsertFromSync(in models.Listing, clientVersion int, partial bool) (models.Listing, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cur, ok := s.byID[in.ID]
	if ok && !cur.Deleted {
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
			if in.ProduceName != "" {
				cur.ProduceName = in.ProduceName
			}
			if in.QuantityListed != 0 {
				cur.QuantityListed = in.QuantityListed
			}
			if in.AskingPrice != 0 {
				cur.AskingPrice = in.AskingPrice
			}
			if in.Location != "" {
				cur.Location = in.Location
			}
			if in.Contact != "" {
				cur.Contact = in.Contact
			}
			if in.Status != "" {
				cur.Status = in.Status
			}
			cur.Deleted = in.Deleted
		}
		cur.Version++
		cur.UpdatedAt = time.Now()
		s.byID[in.ID] = cur
		if storage.DB != nil {
			_ = storage.SaveListing(cur)
		}
		return cur, nil
	}

	if in.ID == "" {
		in.ID = fmt.Sprintf("l-%d", time.Now().UnixNano())
	}
	now := time.Now()
	in.Version = 1
	in.CreatedAt = now
	in.UpdatedAt = now
	if in.Status == "" {
		in.Status = models.StatusAvailable
	}
	s.byID[in.ID] = in
	if storage.DB != nil {
		_ = storage.SaveListing(in)
	}
	return in, nil
}
