package services

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"mavuno/internal/models"
	"mavuno/internal/storage"
)

var ErrNotFound = errors.New("not found")

type ProduceService struct {
	mu   sync.RWMutex
	byID map[string]models.Produce

	conflicts *ConflictService
}

func NewProduceService(conflicts *ConflictService) *ProduceService {
	svc := &ProduceService{byID: make(map[string]models.Produce), conflicts: conflicts}
	// Seed from SQLite on startup
	if storage.DB != nil {
		if rows, err := storage.GetAllProduce(); err == nil {
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

func (s *ProduceService) Patch(id string, clientVersion int, patch map[string]any) (models.Produce, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cur, ok := s.byID[id]
	if !ok || cur.Deleted {
		return models.Produce{}, ErrNotFound
	}
	if s.conflicts != nil {
		if err := s.conflicts.CheckProduce(id, clientVersion, cur, true); err != nil {
			return models.Produce{}, err
		}
	}
	if v, ok := patch["name"].(string); ok {
		cur.ProduceName = v
	}
	if v, ok := patch["category"].(string); ok {
		cur.Category = models.ProduceCategory(v)
	}
	if v, ok := patch["quantity"].(float64); ok {
		cur.Quantity = v
	}
	if v, ok := patch["unit"].(string); ok {
		cur.Unit = v
	}
	if v, ok := patch["price"].(float64); ok {
		cur.PricePerUnit = v
	}
	if v, ok := patch["location"].(string); ok {
		cur.Location = v
	}
	if v, ok := patch["notes"].(string); ok {
		cur.Notes = v
	}
	cur.Version++
	cur.UpdatedAt = time.Now()
	s.byID[id] = cur
	if storage.DB != nil {
		_ = storage.SaveProduce(cur)
	}
	return cur, nil
}

func (s *ProduceService) Delete(id string, clientVersion int) (models.Produce, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cur, ok := s.byID[id]
	if !ok || cur.Deleted {
		return models.Produce{}, ErrNotFound
	}
	if s.conflicts != nil {
		if err := s.conflicts.CheckProduce(id, clientVersion, cur, true); err != nil {
			return models.Produce{}, err
		}
	}
	cur.Deleted = true
	cur.Version++
	cur.UpdatedAt = time.Now()
	s.byID[id] = cur
	if storage.DB != nil {
		_ = storage.SaveProduce(cur)
	}
	return cur, nil
}

func (s *ProduceService) UpsertFromSync(in models.Produce, clientVersion int, partial bool) (models.Produce, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cur, ok := s.byID[in.ID]
	if ok && !cur.Deleted {
		if s.conflicts != nil {
			if err := s.conflicts.CheckProduce(in.ID, clientVersion, cur, true); err != nil {
				return models.Produce{}, err
			}
		}
		if !partial {
			cur = in
		} else {
			if in.ProduceName != "" {
				cur.ProduceName = in.ProduceName
			}
			if in.Category != "" {
				cur.Category = in.Category
			}
			if in.Quantity != 0 {
				cur.Quantity = in.Quantity
			}
			if in.Unit != "" {
				cur.Unit = in.Unit
			}
			if in.PricePerUnit != 0 {
				cur.PricePerUnit = in.PricePerUnit
			}
			if in.Location != "" {
				cur.Location = in.Location
			}
			if in.Notes != "" {
				cur.Notes = in.Notes
			}
			cur.Deleted = in.Deleted
		}
		cur.Version++
		cur.UpdatedAt = time.Now()
		s.byID[in.ID] = cur
		if storage.DB != nil {
			_ = storage.SaveProduce(cur)
		}
		return cur, nil
	}

	if in.ID == "" {
		in.ID = fmt.Sprintf("p-%d", time.Now().UnixNano())
	}
	now := time.Now()
	in.Version = 1
	in.CreatedAt = now
	in.UpdatedAt = now
	s.byID[in.ID] = in
	if storage.DB != nil {
		_ = storage.SaveProduce(in)
	}
	return in, nil
}
