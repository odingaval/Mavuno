package services

import (
	"errors"
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

func NewConflictService() *ConflictService {
	return &ConflictService{}
}
