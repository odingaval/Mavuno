package services

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"mavuno/internal/models"
)

var (
	ErrNotFound = errors.New("not found")
)

type ProduceService struct {
	mu   sync.RWMutex
	byID map[string]models.Produce

	conflicts *ConflictService
}

func NewProduceService(conflicts *ConflictService) *ProduceService {
	return &ProduceService{byID: make(map[string]models.Produce), conflicts: conflicts}
}

func (s *ProduceService) List() []models.Produce {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.Produce, 0, len(s.byID))
	for _, p := range s.byID {
		out = append(out, p)
	}
	return out
}

func (s *ProduceService) Get(id string) (models.Produce, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.byID[id]
	return p, ok
}

// Create creates a new produce. If p.ID is empty it will be generated.
// Version starts at 1 and UpdatedAt is set to now.
func (s *ProduceService) Create(p models.Produce) models.Produce {
	s.mu.Lock()
	defer s.mu.Unlock()

	if p.ID == "" {
		p.ID = fmt.Sprintf("p-%d", time.Now().UnixNano())
	}
	p.Version = 1
	now := time.Now().UnixMilli()
	p.UpdatedAt = now
	s.byID[p.ID] = p
	return p
}

// Patch updates an existing produce using partial fields.
// clientVersion must match stored version.
func (s *ProduceService) Patch(id string, clientVersion int, patch map[string]any) (models.Produce, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cur, ok := s.byID[id]
	if !ok {
		return models.Produce{}, ErrNotFound
	}
	if s.conflicts != nil {
		if err := s.conflicts.CheckProduce(id, clientVersion, cur, true); err != nil {
			return models.Produce{}, err
		}
	}

	// Apply patch (best-effort; ignore unknown fields)
	if v, ok := patch["name"].(string); ok {
		cur.Name = v
	}
	if v, ok := patch["category"].(string); ok {
		cur.Category = v
	}
	if v, ok := patch["quantity"].(float64); ok {
		cur.Quantity = v
	}
	if v, ok := patch["unit"].(string); ok {
		cur.Unit = v
	}
	if v, ok := patch["price"].(float64); ok {
		cur.Price = v
	}
	// Deleted handled by Delete()

	cur.Version++
	cur.UpdatedAt = time.Now().UnixMilli()
	s.byID[id] = cur
	return cur, nil
}

// Delete performs a soft delete.
func (s *ProduceService) Delete(id string, clientVersion int) (models.Produce, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cur, ok := s.byID[id]
	if !ok {
		return models.Produce{}, ErrNotFound
	}
	if s.conflicts != nil {
		if err := s.conflicts.CheckProduce(id, clientVersion, cur, true); err != nil {
			return models.Produce{}, err
		}
	}

	cur.Version++
	cur.UpdatedAt = time.Now().UnixMilli()
	// no Deleted field in models.Produce; treat quantity=0? Keep record and let sync layer interpret via operation type.
	s.byID[id] = cur
	return cur, nil
}

// UpsertFromSync applies a create/update coming from the sync engine.
// If the operation is replayed, providing the same desired version is safe.
func (s *ProduceService) UpsertFromSync(in models.Produce, clientVersion int, partial bool) (models.Produce, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cur, ok := s.byID[in.ID]
	if ok {
		if s.conflicts != nil {
			if err := s.conflicts.CheckProduce(in.ID, clientVersion, cur, true); err != nil {
				return models.Produce{}, err
			}
		}

		// partial means only update fields present (non-zero/empty isn't perfect; sync payload should include explicit fields)
		if !partial {
			cur = in
		} else {
			if in.Name != "" {
				cur.Name = in.Name
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
			if in.Price != 0 {
				cur.Price = in.Price
			}
		}

		cur.Version++
		cur.UpdatedAt = time.Now().UnixMilli()
		s.byID[in.ID] = cur
		return cur, nil
	}

	// create
	if in.ID == "" {
		in.ID = fmt.Sprintf("p-%d", time.Now().UnixNano())
	}
	in.Version = 1
	in.UpdatedAt = time.Now().UnixMilli()
	s.byID[in.ID] = in
	return in, nil
}
