package services

import (
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

type SyncOperation struct {
	OperationID   string     `json:"operation_id"`
	Type          SyncOpType `json:"type"`
	EntityID      string     `json:"entity_id"`
	ClientVersion int        `json:"client_version"`
	Partial       bool       `json:"partial"`
	ClientTime    *time.Time `json:"client_time,omitempty"`

	Produce *models.Produce `json:"produce,omitempty"`
	Listing *models.Listing `json:"listing,omitempty"`
	Patch   map[string]any  `json:"patch,omitempty"`
}

type SyncRequest struct {
	LastSyncedAt *time.Time      `json:"last_synced_at,omitempty"`
	Operations   []SyncOperation `json:"operations"`
}

type OperationResult struct {
	OperationID   string     `json:"operation_id"`
	Type          SyncOpType `json:"type"`
	EntityID      string     `json:"entity_id"`
	Status        string     `json:"status"` // processed | duplicate | conflict | failed
	Error         string     `json:"error,omitempty"`
	ServerVersion int        `json:"server_version,omitempty"`
}

type ConflictResult struct {
	OperationID string         `json:"operation_id"`
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
		produces:  produces,
		listings:  listings,
		conflicts: conflicts,
		ops:       make(map[string]opRecord),
	}
}
