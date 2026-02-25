package models

// BaseModel supports versioning for sync/conflict resolution
type BaseModel struct {
	ID        string `json:"id"`
	Version   int    `json:"version"`
	UpdatedAt int64  `json:"updated_at"`
}
