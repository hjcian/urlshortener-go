package concurrentqueue

import (
	"errors"
	"sync"
)

var (
	ErrEmpty = errors.New("queue is empty")
)

type Queue interface {
	BatchEnqueue(ids []string)
	Dequeue() (string, error)
}

func New() Queue {
	return &filo{}
}

type filo struct {
	mu    sync.RWMutex
	queue []string
}

func (c *filo) BatchEnqueue(ids []string) {
	c.mu.Lock()
	c.queue = append(c.queue, ids...)
	c.mu.Unlock()
}

func (c *filo) Dequeue() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	len := len(c.queue)
	if len == 0 {
		return "", ErrEmpty
	}

	ret := c.queue[len-1]
	c.queue = c.queue[0 : len-1]
	return ret, nil
}