package models

import (
	"time"

	"gorm.io/gorm"
)

type Url struct {
	Id        string `gorm:"primaryKey"`
	Url       string
	ExpiredAt time.Time `gorm:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
