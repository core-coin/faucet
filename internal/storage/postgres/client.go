package postgres

import (
	"git.energy/corepass/corepass-http-helpers/logger"
	"github.com/core-coin/faucet/internal/models"
	"gorm.io/gorm"
)

type postgres struct {
	*logger.Logger

	client *gorm.DB
}

func NewPostgresStorage(logger *logger.Logger, db *gorm.DB) *postgres {
	return &postgres{
		Logger: logger,
		client: db,
	}
}

func (p *postgres) Migrate() error {
	return p.client.AutoMigrate(models.CoreID{})
}
