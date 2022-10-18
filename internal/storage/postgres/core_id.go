package postgres

import (
	"github.com/core-coin/faucet/internal/models"
	"github.com/core-coin/faucet/internal/storage"
	"gorm.io/gorm"
)

func (p *postgres) GetCoreID(id string) (*models.CoreID, error) {
	var coreID *models.CoreID
	err := p.client.Where("id = ?", id).First(&coreID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, storage.ErrorNotFound
		}
		return nil, err
	}
	return coreID, nil
}

func (p *postgres) CreateCoreID(coreID *models.CoreID) error {
	return p.client.Create(coreID).Error
}

func (p *postgres) Verify(coreID string) error {
	return p.client.Model(&models.CoreID{}).Where("id = ?", coreID).Update("verified", true).Error
}
