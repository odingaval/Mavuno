package models

// Farmer represents a farmer's profile in the database.
type Farmer struct {
	BaseModel
	FullName string
	Phone    string
	Location string
}
