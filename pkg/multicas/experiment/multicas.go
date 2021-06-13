package multicas

import (
	"sync"
)

type MultiCAS interface {
	// Set will guarantee there is only one of concurrent goroutines can set successfully.
	Set(key interface{}) bool
	Unset(key interface{})
}

// newMultiCAS_v1_forTest returns a version 1 of MultiCAS.
//
// version 1 uses sync.Mutex as a baseline implementation.
func newMultiCAS_v1_forTest() MultiCAS {
	return &multicas_v1{
		table: make(map[interface{}]bool),
	}
}

type multicas_v1 struct {
	mu    sync.Mutex
	table map[interface{}]bool
}

func (m *multicas_v1) Set(key interface{}) (ok bool) {
	m.mu.Lock()
	if !m.table[key] {
		m.table[key] = true
		ok = true
	}
	m.mu.Unlock()
	return ok
}

func (m *multicas_v1) Unset(key interface{}) {
	m.mu.Lock()
	if m.table[key] {
		delete(m.table, key)
	}
	m.mu.Unlock()
}

// NewMultiCAS_v2 returns a version 2 of MultiCAS.
//
// version 2 uses sync.RWMutex to improve Set() performance around 10% compare to version 1.
func newMultiCAS_v2_forTest() MultiCAS {
	return &multicas_v2{
		table: make(map[interface{}]bool),
	}
}

type multicas_v2 struct {
	mu    sync.RWMutex
	table map[interface{}]bool
}

func (m *multicas_v2) Set(key interface{}) (ok bool) {
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

func (m *multicas_v2) Unset(key interface{}) {
	m.mu.Lock()
	if m.table[key] {
		delete(m.table, key)
	}
	m.mu.Unlock()
}
