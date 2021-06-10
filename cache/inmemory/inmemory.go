package inmemory

import (
	"goshorturl/cache/engine"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

//
// Default in-memory cache engine
//

func New(defaultExp, defaultClearInterval time.Duration) engine.Engine {
	return &inMemory{
		engine: gocache.New(defaultExp, defaultClearInterval),
	}
}

type inMemory struct {
	engine *gocache.Cache
}

func (i *inMemory) Get(id string) (*engine.Entry, bool) {
	data, found := i.engine.Get(id)
	if !found {
		return nil, false
	}
	entry, ok := data.(engine.Entry)
	if !ok {
		// TODO: return additional error for caller to handle?
		return nil, false
	}
	return &entry, true
}

func (i *inMemory) Set(id string, entry *engine.Entry, expiration time.Duration) {
	i.engine.Set(id, *entry, expiration)
}
