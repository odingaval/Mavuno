package services

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"mavuno/internal/models"
)

type SyncOpType string

const (
	OpProduceUpsert SyncOpType = "produce_upsert"
	OpProduceDelete SyncOpType = "produce_delete"
	OpListingUpsert SyncOpType = "listing_upsert"
	OpListingDelete SyncOpType = "listing_delete"
)

// SyncOperation represents a single queued offline write.
type SyncOperation struct {
	OperationID  string     `json:"operation_id"`
	Type         SyncOpType `json:"type"`
	EntityID     string     `json:"entity_id"`
	ClientVersion int       `json:"client_version"`
	Partial      bool       `json:"partial"`
	ClientTime   *time.Time `json:"client_time,omitempty"`

	Produce  *models.Produce `json:"produce,omitempty"`
	Listing  *models.Listing `json:"listing,omitempty"`
	Patch    map[string]any  `json:"patch,omitempty"`
}

type SyncRequest struct {
	LastSyncedAt *time.Time       `json:"last_synced_at,omitempty"`
	Operations   []SyncOperation  `json:"operations"`
}

type OperationResult struct {
	OperationID string     `json:"operation_id"`
	Type        SyncOpType `json:"type"`
	EntityID    string     `json:"entity_id"`
	Status      string     `json:"status"` // processed | duplicate | conflict | failed
	Error       string     `json:"error,omitempty"`
	ServerVersion int      `json:"server_version,omitempty"`
}

type ConflictResult struct {
	OperationID string        `json:"operation_id"`
	Conflict    *ConflictError `json:"conflict"`
}

type SyncResponse struct {
	ServerTime time.Time `json:"server_time"`

	Processed []OperationResult `json:"processed"`
	Conflicts []ConflictResult  `json:"conflicts"`
	Failed    []OperationResult `json:"failed"`

	ChangedProduces []models.Produce `json:"changed_produces"`
	ChangedListings []models.Listing `json:"changed_listings"`
}

// opRecord is used for in-memory idempotency. In production this should be persisted.
type opRecord struct {
	Result OperationResult
}

type SyncService struct {
	produces  *ProduceService
	listings  *ListingService
	conflicts *ConflictService

	muOps sync.RWMutex
	ops   map[string]opRecord
}

func NewSyncService(produces *ProduceService, listings *ListingService, conflicts *ConflictService) *SyncService {
	return &SyncService{
		produces: produces,
		listings: listings,
		conflicts: conflicts,
		ops:      make(map[string]opRecord),
	}
}

func (s *SyncService) Sync(ctx context.Context, req SyncRequest) (SyncResponse, error) {
	resp := SyncResponse{ServerTime: time.Now()}

	for _, op := range req.Operations {
		select {
		case <-ctx.Done():
			return resp, ctx.Err()
		default:
		}

		if op.OperationID == "" {
			resp.Failed = append(resp.Failed, OperationResult{Type: op.Type, EntityID: op.EntityID, Status: "failed", Error: "missing operation_id"})
			continue
		}

		if stored, ok := s.getOp(op.OperationID); ok {
			dup := stored.Result
			dup.Status = "duplicate"
			resp.Processed = append(resp.Processed, dup)
			continue
		}

		res, conflict, err := s.applyOne(op)
		if conflict != nil {
			resp.Conflicts = append(resp.Conflicts, ConflictResult{OperationID: op.OperationID, Conflict: conflict})
			res.Status = "conflict"
			res.Error = "conflict"
			resp.Processed = append(resp.Processed, res)
			s.putOp(op.OperationID, res)
			continue
		}
		if err != nil {
			res.Status = "failed"
			res.Error = err.Error()
			resp.Failed = append(resp.Failed, res)
			// do not store failed ops as processed; client can retry
			continue
		}

		res.Status = "processed"
		resp.Processed = append(resp.Processed, res)
		s.putOp(op.OperationID, res)
	}

	// partial sync: return only records updated after last_synced_at
	if req.LastSyncedAt != nil {
		ms := req.LastSyncedAt.UnixMilli()
		resp.ChangedProduces = filterProducesUpdatedAfter(s.produces.List(), ms)
		resp.ChangedListings = filterListingsUpdatedAfter(s.listings.List(), ms)
	} else {
		// if no timestamp provided, return nothing (client can explicitly do full download)
		resp.ChangedProduces = []models.Produce{}
		resp.ChangedListings = []models.Listing{}
	}

	return resp, nil
}

func (s *SyncService) applyOne(op SyncOperation) (OperationResult, *ConflictError, error) {
	res := OperationResult{OperationID: op.OperationID, Type: op.Type, EntityID: op.EntityID}

	switch op.Type {
	case OpProduceUpsert:
		if op.Produce == nil {
			return res, nil, errors.New("missing produce")
		}
		p := *op.Produce
		if p.ID == "" {
			p.ID = op.EntityID
		}
		updated, err := s.produces.UpsertFromSync(p, op.ClientVersion, op.Partial)
		if err != nil {
			var ce *ConflictError
			if errors.As(err, &ce) {
				return res, ce, nil
			}
			return res, nil, err
		}
		res.EntityID = updated.ID
		res.ServerVersion = updated.Version
		return res, nil, nil

	case OpProduceDelete:
		id := op.EntityID
		if id == "" && op.Produce != nil {
			id = op.Produce.ID
		}
		if id == "" {
			return res, nil, errors.New("missing entity_id")
		}
		updated, err := s.produces.Delete(id, op.ClientVersion)
		if err != nil {
			var ce *ConflictError
			if errors.As(err, &ce) {
				return res, ce, nil
			}
			return res, nil, err
		}
		res.EntityID = updated.ID
		res.ServerVersion = updated.Version
		return res, nil, nil

	case OpListingUpsert:
		if op.Listing == nil {
			return res, nil, errors.New("missing listing")
		}
		l := *op.Listing
		if l.ID == "" {
			l.ID = op.EntityID
		}
		updated, err := s.listings.UpsertFromSync(l, op.ClientVersion, op.Partial)
		if err != nil {
			var ce *ConflictError
			if errors.As(err, &ce) {
				return res, ce, nil
			}
			return res, nil, err
		}
		res.EntityID = updated.ID
		res.ServerVersion = updated.Version
		return res, nil, nil

	case OpListingDelete:
		id := op.EntityID
		if id == "" && op.Listing != nil {
			id = op.Listing.ID
		}
		if id == "" {
			return res, nil, errors.New("missing entity_id")
		}
		updated, err := s.listings.Delete(id, op.ClientVersion)
		if err != nil {
			var ce *ConflictError
			if errors.As(err, &ce) {
				return res, ce, nil
			}
			return res, nil, err
		}
		res.EntityID = updated.ID
		res.ServerVersion = updated.Version
		return res, nil, nil

	default:
		return res, nil, errors.New("unknown op type")
	}
}

func (s *SyncService) getOp(id string) (opRecord, bool) {
	s.muOps.RLock()
	defer s.muOps.RUnlock()
	r, ok := s.ops[id]
	return r, ok
}

func (s *SyncService) putOp(id string, res OperationResult) {
	s.muOps.Lock()
	defer s.muOps.Unlock()
	s.ops[id] = opRecord{Result: res}
}

func filterProducesUpdatedAfter(in []models.Produce, updatedAfterMs int64) []models.Produce {
	out := make([]models.Produce, 0)
	for _, p := range in {
		if p.UpdatedAt > updatedAfterMs {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt < out[j].UpdatedAt })
	return out
}

func filterListingsUpdatedAfter(in []models.Listing, updatedAfterMs int64) []models.Listing {
	out := make([]models.Listing, 0)
	for _, l := range in {
		if l.UpdatedAt > updatedAfterMs {
			out = append(out, l)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt < out[j].UpdatedAt })
	return out
}
