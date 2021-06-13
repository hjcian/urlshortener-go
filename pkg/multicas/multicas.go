package multicas

import (
	"sync"
)

type MultiCAS interface {
	// Set will guarantee there is only one of concurrent goroutines can set successfully.
	Set(key interface{}) bool
	Unset(key interface{})
}

func NewMultiCAS() MultiCAS {
	return &multicas{
		table: make(map[interface{}]bool),
	}
}

type multicas struct {
	mu    sync.RWMutex
	table map[interface{}]bool
}

func (m *multicas) Set(key interface{}) (ok bool) {
	m.mu.RLock()
	isSet := m.table[key]
	m.mu.RUnlock()
	if isSet {
		return false
	}

	m.mu.Lock()
	if !m.table[key] {
		m.table[key] = true
		ok = true
	}
	m.mu.Unlock()
	return ok
}

func (m *multicas) Unset(key interface{}) {
	m.mu.Lock()
	if m.table[key] {
		delete(m.table, key)
	}
	m.mu.Unlock()
}
