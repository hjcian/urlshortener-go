package repository

import (
	"fmt"

	"goshorturl/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Init(port int, host, dbuser, dbname, password string) (*gorm.DB, error) {
	args := fmt.Sprintf("host=%s port=%v user=%s dbname=%s password=%s TimeZone=Asia/Taipei",
		host, port, dbuser, dbname, password)
	db, err := gorm.Open(postgres.Open(args), &gorm.Config{})

	db.AutoMigrate(&models.Url{})
	return db, err
}
