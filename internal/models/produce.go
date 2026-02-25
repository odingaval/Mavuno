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

// Produce represents a farmer's produce record in the database. It embeds BaseModel meaning every produce record automatically

type Produce struct {
	BaseModel
	FarmerID          string
	Category          ProduceCategory
	ProduceName       string
	Quantity          float64
	QuantitySold      float64
	QuantityRejected  float64
	QuantityRemaining float64
	PricePerUnit      float64
	TotalReceived     float64
	Unit              string
	Notes             string
}
