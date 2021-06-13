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

func (i *inMemory) Get(id string) (*cacher.Entry, bool) {
	data, found := i.engine.Get(id)
	if !found {
		return nil, false
	}
	entry, ok := data.(cacher.Entry)
	if !ok {
		// TODO: return additional error for caller to handle?
		return nil, false
	}
	return &entry, true
}

func (i *inMemory) Set(id string, entry *cacher.Entry, expiration time.Duration) {
	i.engine.Set(id, *entry, expiration)
}

func (i *inMemory) Delete(id string) {
	i.engine.Delete(id)
}

func (i *inMemory) Check(id string) bool {
	return i.mcas.Set(id)
}

func (i *inMemory) Uncheck(id string) {
	i.mcas.Unset(id)
}
