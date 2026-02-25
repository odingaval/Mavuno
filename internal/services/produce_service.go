package services

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type ProduceService struct {
	mu   sync.RWMutex
	byID map[string]Produce
}

func NewProduceService() *ProduceService {
	return &ProduceService{byID: make(map[string]Produce)}
}

func (s *ProduceService) List() []Produce {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Produce, 0, len(s.byID))
	for _, p := range s.byID {
		out = append(out, p)
	}
	return out
}

func (s *ProduceService) Get(id string) (Produce, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.byID[id]
	return p, ok
}

func (s *ProduceService) Create(p Produce) Produce {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.ID == "" {
		p.ID = fmt.Sprintf("p-%d", time.Now().UnixNano())
	}
	p.Version = 1
	p.UpdatedAt = time.Now()
	p.Deleted = false
	s.byID[p.ID] = p
	return p
}

func (s *ProduceService) Update(p Produce) (Produce, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.byID[p.ID]
	if !ok {
		return Produce{}, errors.New("not found")
	}
	p.Version = cur.Version + 1
	p.UpdatedAt = time.Now()
	s.byID[p.ID] = p
	return p, nil
}

func (s *ProduceService) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.byID[id]
	if !ok {
		return errors.New("not found")
	}
	p.Deleted = true
	p.Version = p.Version + 1
	p.UpdatedAt = time.Now()
	s.byID[id] = p
	return nil
}
