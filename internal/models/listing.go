package models

type ListingStatus string

const (
	StatusAvailable ListingStatus = "available"
	StatusSold      ListingStatus = "sold"
	StatusCancelled ListingStatus = "cancelled"
)

// Listing represents a market listing.
type Listing struct {
	BaseModel

	ProduceID   string        `json:"produceId"`
	ProduceName string        `json:"produceName"` // denormalised for offline display
	FarmerID    string        `json:"farmerId"`

	QuantityListed float64       `json:"quantity"`
	AskingPrice    float64       `json:"price"`
	Location       string        `json:"location"`
	Contact        string        `json:"contact"`
	Status         ListingStatus `json:"status"`

	BuyerName     string `json:"buyerName"`
	BuyerContact  string `json:"buyerContact"`
	BuyerLocation string `json:"buyerLocation"`
	Notes         string `json:"notes"`
}
