package storage

import (
	"errors"
	"github.com/core-coin/faucet/internal/models"
)

var (
	ErrorNotFound = errors.New("record not found")
)

type Storage interface {
	CoreIDStorage
}

type CoreIDStorage interface {
	GetCoreID(coreID string) (*models.CoreID, error)
	CreateCoreID(coreID *models.CoreID) error
	Verify(coreID string) error
}
