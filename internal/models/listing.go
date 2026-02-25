package models

// Listing represents a market listing (minimal fields for testing)
type Listing struct {
	ID        string  `json:"id"`
	ProduceID string  `json:"produceId"`
	Quantity  float64 `json:"quantity"`
	Price     float64 `json:"price"`
	Location  string  `json:"location"`
	Version   int     `json:"version"`
}
