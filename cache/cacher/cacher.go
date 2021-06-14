package cacher

import "time"

type Entry struct {
	Url string
	Err error
}

type Engine interface {
	Get(id string) (*Entry, bool)
	Set(id string, entry *Entry, expiration time.Duration)
	Delete(id string)

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
	Check(id string) bool
	// Uncheck will return the permission for given id.
	Uncheck(id string)
}
