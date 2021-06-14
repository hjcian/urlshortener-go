package cache

import (
	"context"
	"goshorturl/cache/cacher"
	"goshorturl/cache/inmemory"
	"goshorturl/cache/redis"
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

type cacheOptions struct {
	engine cacher.Engine
}

type Option struct {
	f func(*cacheOptions)
}

func UseInMemoryCache() Option {
	return Option{
		func(c *cacheOptions) {
			c.engine = inmemory.New(defaultExp, defaultClearInterval)
		}}
}

func UseRedis(host string, port int) Option {
	return Option{
		func(c *cacheOptions) {
			c.engine = redis.New(host, port)
		}}
}

func New(db repository.Repository, logger *zap.Logger, options ...Option) repository.Repository {
	opts := cacheOptions{}
	UseInMemoryCache().f(&opts)

	for _, option := range options {
		option.f(&opts)
	}

	return &cacheLogic{
		db:     db,
		logger: logger,
		cache:  opts.engine,
	}
}

type cacheLogic struct {
	db     repository.Repository
	logger *zap.Logger
	cache  cacher.Engine
}

// Get caches results that retrieved from database.
func (r *cacheLogic) Get(ctx context.Context, id string) (string, error) {
	cached, found, err := r.cache.Get(id)
	if err != nil && err != cacher.ErrEntryNotFound {
		r.logger.Warn("cache error", zap.Error(err))
		return "", err
	}
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
	checked, err := r.cache.Check(id)
	if err != nil {
		r.logger.Warn("cache check id error", zap.Error(err), zap.String("id", id))
	}
	if checked {
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
	err = r.cache.Delete(id)
	if err != nil {
		r.logger.Warn("delete cache fail", zap.Error(err), zap.String("id", id))
	}
	return nil
}

// Create adds an entry to cache if that entry is successfully inserted into storage.
func (r *cacheLogic) Create(ctx context.Context, id, url string, expiredAt time.Time) error {
	err := r.db.Create(ctx, id, url, expiredAt)
	if err != nil {
		return err
	}
	exp := time.Until(expiredAt)
	err = r.cache.Set(id, &cacher.Entry{Url: url, Err: err}, exp)
	if err != nil {
		r.logger.Warn("create cache fail", zap.Error(err), zap.String("id", id))
	}
	return nil
}

// Update adds an entry to cache if that entry is successfully updated into storage.
func (r *cacheLogic) Update(ctx context.Context, id, url string, expiredAt time.Time) error {
	err := r.db.Update(ctx, id, url, expiredAt)
	if err != nil {
		return err
	}
	exp := time.Until(expiredAt)
	r.logger.Debug("update cache", zap.String("id", id), zap.String("url", url), zap.Error(err), zap.Any("exp", exp))
	err = r.cache.Set(id, &cacher.Entry{Url: url, Err: err}, exp)
	if err != nil {
		r.logger.Warn("update cache fail", zap.Error(err), zap.String("id", id), zap.String("url", url))
	}
	return err
}

// SelectDeletedAndExpired just wraps the db.SelectDeletedAndExpired().
func (r *cacheLogic) SelectDeletedAndExpired(ctx context.Context, limit int) ([]string, error) {
	return r.db.SelectDeletedAndExpired(ctx, limit)
}
