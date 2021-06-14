package cacher

import (
	"errors"
	"time"
)

var (
	ErrEntryNotFound   = errors.New("entry not found")
	ErrSerializeFailed = errors.New("serialize failed")
)

type Entry struct {
	Url string
	Err error
}

type Engine interface {
	Get(id string) (*Entry, bool, error)
	Set(id string, entry *Entry, expiration time.Duration) error
	Delete(id string) error

	// Check is used for multiple goroutines try to get the access permission
	// for given id.
	//
	// Return `true` means current goroutine get the permission, and has
	// responsibility to call Uncheck();
	//
	// Return `false` means that an another goroutine has already took the
	// permission away.
	//
	// This method is goroutine-safe.
	Check(id string) (bool, error)
	// Uncheck will return the permission for given id.
	Uncheck(id string) error
}
