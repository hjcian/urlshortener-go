package idgenerator

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"goshorturl/pkg/concurrentstack"
	"goshorturl/repository"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jxskiss/base62"
	"go.uber.org/zap"
)

const (
	totalLetters     = 6
	encodedChars     = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	selectAll        = -1
	doRecycleTimeout = 30 * time.Second
)

type empty struct{}

var validCharSet map[rune]empty

var (
	encoder           = base62.NewEncoding(encodedChars)
	errInvalidLength  = errors.New("invalid length")
	errUnexpectedChar = errors.New("unexpected char")
)

// generate returns a 6-letters id by given URL.
func generate(url string) string {
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

type IDGenerator interface {
	Get(ctx context.Context, url string, expiredAt time.Time) (string, error)
}

func New(db repository.Repository, logger *zap.Logger) IDGenerator {
	return &idGenerator{
		db:     db,
		logger: logger,
		ids:    concurrentstack.New(),
	}
}

type idGenerator struct {
	db          repository.Repository
	logger      *zap.Logger
	ids         concurrentstack.Stack
	doRecycling int32
}

func (i *idGenerator) Get(ctx context.Context, url string, expiredAt time.Time) (string, error) {
	id, err := i.ids.Pop()
	if err != concurrentstack.ErrEmpty {
		i.logger.Debug("get id from pool", zap.String("id", id))
		err := i.db.Update(ctx, id, url, expiredAt)
		if err != nil {
			i.logger.Error("refresh id with new meta error", zap.Error(err))
			i.ids.Push(id)
			return "", err
		}
		return id, nil
	}
	go i.recycleID(ctx)

	// create a new id
	id = generate(url)
	if err := i.db.Create(ctx, id, url, expiredAt); err != nil {
		i.logger.Error("create new record error", zap.Error(err))
		return "", err
	}
	return id, nil
}

// RecycleID will guarantee only one goroutine can trigger recycling process
func (i *idGenerator) recycleID(ctx context.Context) {
	if atomic.CompareAndSwapInt32(&i.doRecycling, 0, 1) {
		i.logger.Debug("trigger recycling process")

		ctxWithDealine, cancel := context.WithTimeout(ctx, doRecycleTimeout)
		defer cancel()

		ids, err := i.db.SelectDeletedAndExpired(ctxWithDealine, selectAll)
		if err != nil && err != repository.ErrRecordNotFound {
			i.logger.Error("recycle deleted ids error", zap.Error(err))
			i.doRecycling = 0
			return
		}
		i.logger.Debug("recycled ids", zap.Int("count", len(ids)), zap.String("ids", strings.Join(ids, " | ")))
		if len(ids) > 0 {
			i.ids.BatchPush(ids)
		}
		i.doRecycling = 0
	}
}
