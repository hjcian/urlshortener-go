package cache

import (
	"context"
	"fmt"
	"goshorturl/repository"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

const exampleURL = "http://example.com"

type dbRecorder struct {
	repository.UnimplementedRepository
	mutex       sync.Mutex
	getCount    int
	deleteCount int
}

func (d *dbRecorder) Get(ctx context.Context, id string) (string, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.getCount++
	return exampleURL, nil
}
func (d *dbRecorder) Delete(ctx context.Context, id string) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.deleteCount++
	return nil
}

func Test_Cache_Get(t *testing.T) {
	t.Run("same id should only one goroutine can go to database", func(t *testing.T) {
		r := dbRecorder{}
		cache := New(&r, zap.NewNop())
		ctx := context.Background()

		numG := 200000
		var wg sync.WaitGroup
		wg.Add(numG)
		for i := 0; i < numG; i++ {
			go func() {
				defer wg.Done()
				cache.Get(ctx, "aaaaaa")
			}()
		}
		wg.Wait()

		assert.Equal(t, 1, r.getCount)
	})
	t.Run("diffrent id will go to database only once and hit cache afterwards", func(t *testing.T) {
		r := dbRecorder{}
		cache := New(&r, zap.NewNop())
		ctx := context.Background()

		concurrentCall := func(numG int) {
			var wg sync.WaitGroup
			wg.Add(numG)
			for i := 0; i < numG; i++ {
				go func(i int) {
					defer wg.Done()
					cache.Get(ctx, fmt.Sprintln(i))
				}(i)
			}
			wg.Wait()
		}
		numG := 100000
		concurrentCall(numG) // first call
		assert.Equal(t, numG, r.getCount)
		concurrentCall(numG) // second call
		assert.Equal(t, numG, r.getCount)
	})
}

func Test_Cache_Delete(t *testing.T) {
	t.Run("should hit database", func(t *testing.T) {
		r := dbRecorder{}
		cache := New(&r, zap.NewNop())

		err := cache.Delete(context.Background(), "aaaaaa")
		assert.NoError(t, err)
		assert.Equal(t, 1, r.deleteCount)
	})
}
