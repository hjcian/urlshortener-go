package cache

import (
	"context"
	"goshorturl/cache/cacher"
	"goshorturl/cache/inmemory"
	"goshorturl/repository"
	"time"

	"go.uber.org/zap"
)

const (
	defaultClearInterval = 24 * time.Hour
	defaultExp           = 1 * time.Hour
	validEntryExp        = 24 * time.Hour
	emptyEntryExp        = 1 * time.Hour
)

func New(db repository.Repository, logger *zap.Logger) repository.Repository {
	return &cacheLogic{
		db:     db,
		logger: logger,
		cache:  inmemory.New(defaultExp, defaultClearInterval),
	}
}

type cacheLogic struct {
	db     repository.Repository
	logger *zap.Logger
	cache  cacher.Engine
}

// Get caches results that retrieved from database.
func (r *cacheLogic) Get(ctx context.Context, id string) (string, error) {
	cached, found := r.cache.Get(id)
	if found {
		r.logger.Debug(
			"found cached record",
			zap.String("id", id),
			zap.String("url", cached.Url),
			zap.Error(cached.Err))
		return cached.Url, cached.Err
	}

	// cache miss
	r.logger.Debug("cache missed", zap.String("id", id))
	// TODO: use bloomfilter to filter out the non-existed key to reduce the
	// caching load
	if r.cache.Check(id) {
		defer r.cache.Uncheck(id)
		// To avoid cache stampede, Check() ensures that only one goroutine
		// able to trigger cache recomputation until that process finished.
		r.logger.Debug("recompute cache", zap.String("id", id))
		url, err := r.db.Get(ctx, id)
		exp := validEntryExp
		if err != nil {
			exp = emptyEntryExp
		}
		r.cache.Set(id, &cacher.Entry{Url: url, Err: err}, exp)
		return url, err
	}
	// In case of cache stampede, this implementation choose to guarantee
	// the availability, so just return record not found
	return "", repository.ErrRecordNotFound
}

// Delete deletes the record from storage and cache.
func (r *cacheLogic) Delete(ctx context.Context, id string) error {
	// TODO: use bloomfilter to filter out the non-existed key to prevent
	// 		 malicious calls hitting database
	err := r.db.Delete(ctx, id)
	if err != nil {
		return err
	}
	r.logger.Debug("delete cache", zap.String("id", id))
	r.cache.Delete(id)
	return nil
}

// Create adds an entry to cache if that entry is successfully inserted into storage.
func (r *cacheLogic) Create(ctx context.Context, id, url string, expiredAt time.Time) error {
	err := r.db.Create(ctx, id, url, expiredAt)
	if err != nil {
		return err
	}
	exp := time.Until(expiredAt)
	r.cache.Set(id, &cacher.Entry{Url: url, Err: err}, exp)
	return nil
}

// Update adds an entry to cache if that entry is successfully updated into storage.
func (r *cacheLogic) Update(ctx context.Context, id, url string, expiredAt time.Time) error {
	err := r.db.Update(ctx, id, url, expiredAt)
	if err != nil {
		return err
	}
	exp := time.Until(expiredAt)
	r.cache.Set(id, &cacher.Entry{Url: url, Err: err}, exp)
	return nil
}

// SelectDeletedAndExpired just wraps the db.SelectDeletedAndExpired().
func (r *cacheLogic) SelectDeletedAndExpired(ctx context.Context, limit int) ([]string, error) {
	return r.db.SelectDeletedAndExpired(ctx, limit)
}
