package services

import (
	"errors"
)

var ErrConflict = errors.New("conflict")

type ConflictError struct {
	Entity        string
	ID            string
	ServerVersion int
	ServerData    interface{}
}

func (e *ConflictError) Error() string { return "conflict" }

type ConflictService struct{}

func NewConflictService() *ConflictService { return &ConflictService{} }

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
