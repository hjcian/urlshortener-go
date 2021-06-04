package models

import "time"

type Url struct {
	Key       string `gorm:"primaryKey"`
	Url       string
	ExpiredAt time.Time `gorm:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time `gorm:"index"`
}
