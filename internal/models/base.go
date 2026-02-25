package models

import "time"

// BaseModel contains common fields shared by all models in the database.
type BaseModel struct {
	ID        string
	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
	Deleted   bool
}