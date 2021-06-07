package repository

import (
	"context"
	"fmt"
	"time"

	"goshorturl/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitPGRepo(port int, host, dbuser, dbname, password string) (Repository, error) {
	args := fmt.Sprintf("host=%s port=%v user=%s dbname=%s password=%s TimeZone=Asia/Taipei",
		host, port, dbuser, dbname, password)
	db, err := gorm.Open(postgres.Open(args), &gorm.Config{})

	db.AutoMigrate(&models.Url{})
	return &postgresRepository{db: db}, err
}

func InitPGRepoWith(dial gorm.Dialector, cfg gorm.Config) (Repository, error) {
	db, err := gorm.Open(dial, &cfg)

	db.AutoMigrate(&models.Url{})
	return &postgresRepository{db: db}, err
}

type postgresRepository struct {
	db *gorm.DB
}

func (p *postgresRepository) Create(ctx context.Context, id, url string, expiredAt time.Time) error {
	urlEntry := models.Url{
		Id:        id,
		Url:       url,
		ExpiredAt: expiredAt,
	}
	return p.db.Create(&urlEntry).Error
}

func (p *postgresRepository) Delete(ctx context.Context, id string) error {
	res := p.db.Delete(&models.Url{Id: id})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return ErrRecordNotFound
	}
	return nil
}

func (p *postgresRepository) Get(ctx context.Context, id string) (string, error) {
	var result models.Url
	if err := p.db.Where(
		// REMINDER: GORM will use `"urls"."deleted_at" IS NULL` to filter the deleted record
		"id = ? AND expired_at > ?",
		id, time.Now(),
	).Take(&result).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", ErrRecordNotFound
		}
		return "", err
	}
	return result.Url, nil
}
