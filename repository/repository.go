package repository

import (
	"context"
	"errors"
	"time"
)

var (
	ErrRecordNotFound = errors.New("record not found")
)

type Repository interface {
	Create(ctx context.Context, id, url string, expiredAt time.Time) error
	Update(ctx context.Context, id, url string, expiredAt time.Time) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (string, error)
	SelectDeletedAndExpired(ctx context.Context, limit int) ([]string, error)
}

type UnimplementedRepository struct{}

func (u *UnimplementedRepository) Create(ctx context.Context, id, url string, expiredAt time.Time) error {
	return nil
}

func (u *UnimplementedRepository) Update(ctx context.Context, id, url string, expiredAt time.Time) error {
	return nil
}

func (u *UnimplementedRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func (u *UnimplementedRepository) SelectDeletedAndExpired(ctx context.Context, limit int) ([]string, error) {
	return nil, nil
}

func (u *UnimplementedRepository) Get(ctx context.Context, id string) (string, error) {
	return "", nil
}
