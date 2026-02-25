package services

type ConflictService struct{}

func NewConflictService() *ConflictService {
	return &ConflictService{}
}

func (c *ConflictService) ResolveProduceConflict(local, server Produce) Produce {
	if local.Version > server.Version {
		return local
	} else if server.Version > local.Version {
		return server
	} else {
		if local.UpdatedAt.After(server.UpdatedAt) {
			return local
		}
		return server
	}
}

func (c *ConflictService) ResolveListingConflict(local, server Listing) Listing {
	if local.Version > server.Version {
		return local
	} else if server.Version > local.Version {
		return server
	} else {
		if local.UpdatedAt.After(server.UpdatedAt) {
			return local
		}
		return server
	}
}
