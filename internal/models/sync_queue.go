package models

import "time"

// SyncOperation defines the type of operation waiting to be synced.
type SyncOperation string

// SyncEntity defines which table the operation belongs to.
type SyncEntity string

// SyncStatus defines the current state of a sync queue item.
type SyncStatus string

const (
	// Operations
	OperationCreate SyncOperation = "CREATE"
	OperationUpdate SyncOperation = "UPDATE"
	OperationDelete SyncOperation = "DELETE"

	// Entities
	EntityProduce SyncEntity = "produce"
	EntityListing SyncEntity = "listing"
	EntityFarmer  SyncEntity = "farmer"

	// Statuses
	StatusPending SyncStatus = "pending" // Waiting to be synced
	StatusSynced  SyncStatus = "synced"  // Successfully synced
	StatusFailed  SyncStatus = "failed"  // Failed after max retries
)

// SyncQueue represents a pending operation waiting to be sent to the server
// Every time a farmer makes a change offline, a record is added here
// The sync engine processes these records when internet is available
type SyncQueue struct {
	ID          string
	EntityType  SyncEntity
	Operation   SyncOperation
	Payload     string
	Status      SyncStatus
	RetryCount  int
	LastAttempt time.Time
	CreatedAt   time.Time
}
