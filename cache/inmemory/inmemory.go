package inmemory

import (
	"goshorturl/cache/cacher"
	"goshorturl/pkg/multicas"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

// New returns an in-memory cache for default usage.
func New(defaultExp, defaultClearInterval time.Duration) cacher.Engine {
	return &inMemory{
		engine: gocache.New(defaultExp, defaultClearInterval),
		mcas:   multicas.NewMultiCAS(),
	}
}

type inMemory struct {
	engine *gocache.Cache
	mcas   multicas.MultiCAS
}

func (i *inMemory) Get(id string) (*cacher.Entry, bool, error) {
	data, found := i.engine.Get(id)
	if !found {
		return nil, false, cacher.ErrEntryNotFound
	}
	entry, ok := data.(cacher.Entry)
	if !ok {
		return nil, false, cacher.ErrSerializeFailed
	}
	return &entry, true, nil
}

func (i *inMemory) Set(id string, entry *cacher.Entry, expiration time.Duration) error {
	i.engine.Set(id, *entry, expiration)
	return nil
}

func (i *inMemory) Delete(id string) error {
	i.engine.Delete(id)
	return nil
}

func (i *inMemory) Check(id string) (bool, error) {
	return i.mcas.Set(id), nil
}

func (i *inMemory) Uncheck(id string) error {
	i.mcas.Unset(id)
	return nil
}
