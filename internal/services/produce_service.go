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
