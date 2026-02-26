package models

// ProduceCategory defines the type of produce a farmer can record.
type ProduceCategory string

const (
	CategoryDairy     ProduceCategory = "dairy"
	CategoryPoultry   ProduceCategory = "poultry"
	CategoryLivestock ProduceCategory = "livestock"
	CategoryCrops     ProduceCategory = "crops"
	CategoryOther     ProduceCategory = "other"
)

// Produce represents a farmer's produce record in the database.
type Produce struct {
	BaseModel

	FarmerID          string          `json:"farmerId"`
	Category          ProduceCategory `json:"category"`
	ProduceName       string          `json:"name"`
	Quantity          float64         `json:"quantity"`
	QuantitySold      float64         `json:"quantitySold"`
	QuantityRejected  float64         `json:"quantityRejected"`
	QuantityRemaining float64         `json:"quantityRemaining"`
	PricePerUnit      float64         `json:"price"`
	TotalReceived     float64         `json:"totalReceived"`
	Unit              string          `json:"unit"`
	Location          string          `json:"location"`
	Notes             string          `json:"notes"`
}
