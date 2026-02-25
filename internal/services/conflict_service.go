package services

type ConflictService struct { // Handles version/timestamp conflicts
}

func NewConflictService() *ConflictService { // Creates a new ConflictService instance
	return &ConflictService{}
}

func (c *ConflictService) ResolveProduceConflict(local, server Produce) Produce { // Decides which Produce to keep based on version/timestamp
	return server
}

func (c *ConflictService) ResolveListingConflict(local, server Listing) Listing { // Decides which Listing to keep based on version/timestamp
	return server
}
