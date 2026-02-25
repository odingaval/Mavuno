package models

// ListingStatus defines the current state of a market listing.This tracks where in the selling process a listing currently is.
type ListingStatus string

const (
	StatusAvailable   ListingStatus = "available"
	StatusNegotiating ListingStatus = "negotiating"
	StatusSold        ListingStatus = "sold"
)

// Listing represents a farmer's market listing in the database. It embeds BaseModel
type Listing struct {
	BaseModel
	ProduceID      string
	FarmerID       string
	QuantityListed float64
	AskingPrice    float64
	Location       string
	Status         ListingStatus
	BuyerName      string
	BuyerContact   string
	BuyerLocation  string
	Notes          string
}
