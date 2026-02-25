package services

import (
	"time"
)

type BaseModel struct {
	ID        string    `json:"id"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updatedAt"`
	Deleted   bool      `json:"deleted"`
}

type Produce struct {
	BaseModel
	Name    string `json:"name"`
	Quality int    `json:"quality"`
}

type Listing struct {
	BaseModel
	ProduceID string  `json:"produceId"`
	Price     float64 `json:"price"`
}

type SyncRequest struct { // What the client sends to the server when requesting a sync
	LastSync time.Time `json:"lastSync"`
	Produces []Produce `json:"produces"`
	Listings []Listing `json:"listings"`
}

type SyncResponse struct { // What the server returns to the client after processing the sync request
	ServiceTime time.Time `json:"serviceTime"`
}

type SyncService struct {
	conflictService *ConflictService
}

func NewSyncService(conflictService *ConflictService) *SyncService {
	return &SyncService{conflictService: conflictService}
}

func (s *SyncService) Sync(request SyncRequest) SyncResponse {
	return SyncResponse{
		ServiceTime: time.Now(),
	}
}
