package services

import (
	"errors"
	"sync"

	"mavuno/internal/models"
)

var ErrNotFound = errors.New("not found")

type ProduceService struct {
	mu   sync.RWMutex
	byID map[string]models.Produce

	conflicts *ConflictService
}
