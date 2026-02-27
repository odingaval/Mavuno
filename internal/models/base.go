package models

import "time"

// BaseModel contains common fields shared by all models in the database.
type BaseModel struct {
	ID        string    `json:"id"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Deleted   bool      `json:"deleted"`
}
