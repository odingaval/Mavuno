package services

type ConflictService struct { // Handles version/timestamp conflicts
}

func NewConflictService() *ConflictService { // Creates a new ConflictService instance
	return &ConflictService{}
}

func (c *ConflictService) ResolveProduceConflict(local, server Produce) Produce { // Decides which Produce to keep based on version/timestamp
	if local.Version > server.Version {
		return local
	} else if server.Version > local.Version {
		return server
	} else {
		if local.UpdatedAt.After(server.UpdatedAt) { // If versions are equal, use timestamp to decide
			return local
		}
		return server
	}
}

func (c *ConflictService) ResolveListingConflict(local, server Listing) Listing { // Decides which Listing to keep based on version/timestamp
	if local.Version > server.Version {
		return local
	} else if server.Version > local.Version {
		return server
	} else {
		if local.UpdatedAt.After(server.UpdatedAt) { // If versions are equal, use timestamp to decide
			return local
		}
		return server
	}
}
