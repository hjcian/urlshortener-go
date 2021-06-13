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

	// Check checks whether a goroutine has get the access permission for given id.
	//
	// This method is goroutine-safe.
	Check(id string) bool
	// Uncheck will return the permission for given id.
	Uncheck(id string)
}
