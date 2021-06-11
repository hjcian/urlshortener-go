package concurrentstack

import (
	"errors"
	"sync"
)

var (
	ErrEmpty = errors.New("stack is empty")
)

type Stack interface {
	Push(id string)
	BatchPush(ids []string)
	Pop() (string, error)
	Len() int
}

func New() Stack {
	return &filo{
		stack: make([]string, 0),
	}
}

type filo struct {
	mu    sync.RWMutex
	stack []string
}

func (c *filo) Push(id string) {
	c.mu.Lock()
	c.stack = append(c.stack, id)
	c.mu.Unlock()
}

func (c *filo) BatchPush(ids []string) {
	c.mu.Lock()
	c.stack = append(c.stack, ids...)
	c.mu.Unlock()
}

func (c *filo) Pop() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	len := len(c.stack)
	if len == 0 {
		return "", ErrEmpty
	}

	ret := c.stack[len-1]
	c.stack = c.stack[:len-1]
	return ret, nil
}

func (c *filo) Len() int {
	c.mu.RLock()
	len := len(c.stack)
	c.mu.RUnlock()
	return len
}
