package cache

import (
	"context"
	"goshorturl/cache/cacher"
	"goshorturl/cache/inmemory"
	"goshorturl/pkg/multicas"
	"goshorturl/repository"
	"time"
)

const (
	defaultClearInterval = 24 * time.Hour
	defaultExp           = 1 * time.Hour
	cacheHitExp          = 24 * time.Hour
	cacheMissExp         = 1 * time.Hour
)

func New(db repository.Repository) repository.Repository {
	return &cacheLogic{
		db:    db,
		cache: inmemory.New(defaultExp, defaultClearInterval),
		mcas:  multicas.NewMultiCAS(),
	}
}

type cacheLogic struct {
	db    repository.Repository
	cache cacher.Engine
	mcas  multicas.MultiCAS
}

// Get caches results that retrieved from database.
func (r *cacheLogic) Get(ctx context.Context, id string) (string, error) {
	cached, found := r.cache.Get(id)
	if found {
		return cached.Url, cached.Err
	}

	// cache miss
	// TODO: use bloomfilter to filter out the non-existed key to reduce the caching load
	// TODO: use redis's feature to do this logic, instead of
	// 		 using self-implemented `multicas.MultiCAS`
	if r.mcas.Set(id) {
		defer r.mcas.Unset(id)
		// In case of cache stampede, mcas.Set() ensures that only allow
		// one goroutine can trigger recompute the value by id.
		url, err := r.db.Get(ctx, id)
		exp := cacheHitExp
		if err != nil {
			exp = cacheMissExp
		}
		r.cache.Set(id, &cacher.Entry{Url: url, Err: err}, exp)
		return url, err
	}
	//
	// In case of cache stampede, this implementation choose to guarantee the availability,
	// so just return record not found
	return "", repository.ErrRecordNotFound
}

// Create just wraps the db.Create().
func (r *cacheLogic) Create(ctx context.Context, id, url string, expiredAt time.Time) error {
	return r.db.Create(ctx, id, url, expiredAt)
}

// Update just wraps the db.Update().
func (r *cacheLogic) Update(ctx context.Context, id, url string, expiredAt time.Time) error {
	return r.db.Update(ctx, id, url, expiredAt)
}

// Delete just wraps the db.Delete().
func (r *cacheLogic) Delete(ctx context.Context, id string) error {
	return r.db.Delete(ctx, id)
}

// SelectDeletedAndExpired just wraps the db.SelectDeletedAndExpired().
func (r *cacheLogic) SelectDeletedAndExpired(ctx context.Context, limit int) ([]string, error) {
	return r.db.SelectDeletedAndExpired(ctx, limit)
}
