package models

import (
	"gorm.io/gorm"
	"time"
)

type CoreID struct {
	ID       string `json:"id,omitempty" gorm:"primaryKey"`
	Verified bool   `json:"verified"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
