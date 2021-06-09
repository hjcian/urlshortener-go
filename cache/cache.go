package cache

import (
	"context"
	"goshorturl/cache/multicas"
	"goshorturl/repository"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

const (
	defaultClearInterval = 24 * time.Hour
	defaultExp           = 1 * time.Hour
	cacheHitExp          = 24 * time.Hour
	cacheMissExp         = 1 * time.Hour
)

func New(db repository.Repository) repository.Repository {
	return &cacheLogic{
		db: db,
		cache: &inMemory{
			engine: gocache.New(defaultExp, defaultClearInterval),
		},
		mcas: multicas.NewMultiCAS(),
	}
}

type cacheLogic struct {
	db    repository.Repository
	cache cacheEngine
	mcas  multicas.MultiCAS
}

// Create just wraps the db.Create().
func (r *cacheLogic) Create(ctx context.Context, id, url string, expiredAt time.Time) error {
	return r.db.Create(ctx, id, url, expiredAt)
}

// Delete just wraps the db.Delete().
func (r *cacheLogic) Delete(ctx context.Context, id string) error {
	return r.db.Delete(ctx, id)
}

// Get caches results that retrieved from database.
func (r *cacheLogic) Get(ctx context.Context, id string) (string, error) {
	cached, found := r.cache.Get(id)
	if found {
		return cached.url, cached.err
	}

	// cache miss
	// TODO: use redis's feature to do this logic, instead of
	// 		 using self-implemented `multicas.MultiCAS`
	if r.mcas.Set(id) {
		defer r.mcas.Unset(id)
		//
		// In case of cache stampede, mcas.Set() ensures that only allow
		// one goroutine can trigger recompute the value by id.
		// Then release the lock after cache updated
		url, err := r.db.Get(ctx, id)
		exp := cacheHitExp
		if err != nil {
			exp = cacheMissExp
		}
		r.cache.Set(id, &cacheEntry{url, err}, exp)
		return url, err
	}
	//
	// In case of cache stampede, this implementation choose to guarantee the availability,
	// so just return record not found
	return "", repository.ErrRecordNotFound
}

type cacheEntry struct {
	url string
	err error
}

type cacheEngine interface {
	Get(id string) (*cacheEntry, bool)
	Set(id string, entry *cacheEntry, expiration time.Duration)
}

//
// Default in-memory cache engine
//

type inMemory struct {
	engine *gocache.Cache
}

func (i *inMemory) Get(id string) (*cacheEntry, bool) {
	data, found := i.engine.Get(id)
	if !found {
		return nil, false
	}
	entry, ok := data.(cacheEntry)
	if !ok {
		// TODO: return additional error for caller to handle?
		return nil, false
	}
	return &entry, true
}

func (i *inMemory) Set(id string, entry *cacheEntry, expiration time.Duration) {
	i.engine.Set(id, *entry, expiration)
}
