package concurrentqueue

import (
	"errors"
	"sync"
)

var (
	ErrEmpty = errors.New("queue is empty")
)

type Queue interface {
	Enqueue(id string)
	BatchEnqueue(ids []string)
	Dequeue() (string, error)
	Len() int
}

func New() Queue {
	return &filo{
		queue: make([]string, 0),
	}
}

type filo struct {
	mu    sync.RWMutex
	queue []string
}

func (c *filo) Enqueue(id string) {
	c.mu.Lock()
	c.queue = append(c.queue, id)
	c.mu.Unlock()
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
	c.queue = c.queue[:len-1]
	return ret, nil
}

func (c *filo) Len() int {
	c.mu.RLock()
	len := len(c.queue)
	c.mu.RUnlock()
	return len
}
