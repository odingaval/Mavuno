package services

import (
	"errors"

	"mavuno/internal/models"
)

var ErrConflict = errors.New("conflict")

// ConflictError is returned when a client attempts to write using a stale version.
// It includes the current server state so the client can resolve.
type ConflictError struct {
	Entity        string      `json:"entity"`
	ID            string      `json:"id"`
	ServerVersion int         `json:"server_version"`
	ServerData    interface{} `json:"server_data"`
}

func (e *ConflictError) Error() string { return "conflict" }

type ConflictService struct{}

func NewConflictService() *ConflictService { return &ConflictService{} }

// CheckVersion enforces version-based optimistic concurrency.
// If the server record exists, clientVersion must match serverVersion.
func (c *ConflictService) CheckVersion(entity, id string, clientVersion, serverVersion int, serverData interface{}) error {
	if serverVersion == 0 {
		return nil
	}
	if clientVersion != serverVersion {
		return &ConflictError{Entity: entity, ID: id, ServerVersion: serverVersion, ServerData: serverData}
	}
	return nil
}

func (c *ConflictService) CheckProduce(id string, clientVersion int, server models.Produce, exists bool) error {
	if !exists {
		return nil
	}
	return c.CheckVersion("produce", id, clientVersion, server.Version, server)
}

func (c *ConflictService) CheckListing(id string, clientVersion int, server models.Listing, exists bool) error {
	if !exists {
		return nil
	}
	return c.CheckVersion("listing", id, clientVersion, server.Version, server)
}
