package models

// Produce represents a farmer's produce record (minimal fields for testing)
type Produce struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Category  string  `json:"category"`
	Quantity  float64 `json:"quantity"`
	Unit      string  `json:"unit"`
	Price     float64 `json:"price"`
	Version   int     `json:"version"`
	UpdatedAt int64   `json:"updated_at"`
	Deleted   bool    `json:"deleted"`
}
