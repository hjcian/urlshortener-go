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
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (string, error)
}
