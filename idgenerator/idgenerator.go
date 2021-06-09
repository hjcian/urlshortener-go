package idgenerator

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"goshorturl/repository"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jxskiss/base62"
)

const (
	totalLetters = 6
	encodedChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
)

type empty struct{}

var validCharSet map[rune]empty

var (
	encoder           = base62.NewEncoding(encodedChars)
	errInvalidLength  = errors.New("invalid length")
	errUnexpectedChar = errors.New("unexpected char")
)

// Generate returns a 6-letters id by given URL.
//
// TODO: extract this functionality to another stand-alone service
func Generate(url string) string {
	// padding with time.Now().UnixNano() to reduce collision probability if give same URL
	bytes := md5.Sum([]byte(fmt.Sprintf("%s%d", url, time.Now().UnixNano())))
	encoded := encoder.EncodeToString(bytes[:])
	return encoded[:totalLetters]
}

func getValidCharSet() map[rune]empty {
	if validCharSet != nil {
		return validCharSet
	}
	// lazy initialize encodedCharSet
	validCharSet := make(map[rune]empty, len(encodedChars))
	for _, c := range encodedChars {
		validCharSet[c] = empty{}
	}
	return validCharSet
}

func Validate(id string) error {
	if len(id) != totalLetters {
		return errInvalidLength
	}
	validChars := getValidCharSet()
	for _, r := range id {
		if _, ok := validChars[r]; !ok {
			return errUnexpectedChar
		}
	}
	return nil
}

//
//
//

type IDGenerator interface {
	Get(ctx context.Context, url string, expiredAt time.Time) (string, error)
	Validate(id string) error
}

func NewIDGenerator(db repository.Repository) IDGenerator {
	return &idGenerator{
		db:  db,
		idq: &concurrentQueue{},
	}
}

type idGenerator struct {
	db          repository.Repository
	idq         queue
	doRecycling int32
}

func (i *idGenerator) Get(ctx context.Context, url string, expiredAt time.Time) (string, error) {
	id, err := i.idq.Dequeue()
	if err != ErrEmpty {
		// TODO: update DB
		// enqueue the id back if error occurs
		return id, nil
	}
	// call recycle only queue is empty
	i.recycleID()
	// create a new id
	id = Generate(url)
	if err := i.db.Create(ctx, id, url, expiredAt); err != nil {
		return "", err
	}
	return id, nil
}

func (i *idGenerator) Validate(id string) error {
	if len(id) != totalLetters {
		return errInvalidLength
	}
	validChars := getValidCharSet()
	for _, r := range id {
		if _, ok := validChars[r]; !ok {
			return errUnexpectedChar
		}
	}
	return nil
}

// RecycleID will guarantee only one goroutine can trigger recycling process
func (i *idGenerator) recycleID() {
	if atomic.CompareAndSwapInt32(&i.doRecycling, 0, 1) {
		// TODO:
		// select expired and deleted from DB
		//
		ids := []string{}
		i.idq.BatchEnqueue(ids)
		i.doRecycling = 0
	}
}

var (
	ErrEmpty = errors.New("queue is empty")
)

type queue interface {
	BatchEnqueue(ids []string)
	Dequeue() (string, error)
}

type concurrentQueue struct {
	mu    sync.RWMutex
	queue []string
}

func (c *concurrentQueue) BatchEnqueue(ids []string) {
	c.mu.Lock()
	c.queue = append(c.queue, ids...)
	c.mu.Unlock()
}

func (c *concurrentQueue) Dequeue() (string, error) {
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
