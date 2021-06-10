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
}
