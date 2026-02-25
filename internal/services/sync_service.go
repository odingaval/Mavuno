package services

import (
	"time"
)

type SyncRequest struct { // What the client sends to the server when requesting a sync
	LastSync time.Time `json:"lastSync"`
}

type SyncResponse struct { // What the server returns to the client after processing the sync request
	ServiceTime time.Time `json:"serviceTime"`
}
