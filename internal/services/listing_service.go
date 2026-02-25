package services

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type ListingService struct {
	mu   sync.RWMutex
	byID map[string]Listing
}

func NewListingService() *ListingService {
	return &ListingService{byID: make(map[string]Listing)}
}

func (s *ListingService) List() []Listing {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Listing, 0, len(s.byID))
	for _, l := range s.byID {
		out = append(out, l)
	}
	return out
}

func (s *ListingService) Get(id string) (Listing, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	l, ok := s.byID[id]
	return l, ok
}

func (s *ListingService) Create(l Listing) Listing {
	s.mu.Lock()
	defer s.mu.Unlock()
	if l.ID == "" {
		l.ID = fmt.Sprintf("L-%d", time.Now().UnixNano())
	}
	l.Version = 1
	l.UpdatedAt = time.Now()
	l.Deleted = false
	s.byID[l.ID] = l
	return l
}

func (s *ListingService) Update(l Listing) (Listing, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.byID[l.ID]
	if !ok {
		return Listing{}, errors.New("not found")
	}
	l.Version = cur.Version + 1
	l.UpdatedAt = time.Now()
	s.byID[l.ID] = l
	return l, nil
}

func (s *ListingService) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	l, ok := s.byID[id]
	if !ok {
		return errors.New("not found")
	}
	l.Deleted = true
	l.Version = l.Version + 1
	l.UpdatedAt = time.Now()
	s.byID[id] = l
	return nil
}
