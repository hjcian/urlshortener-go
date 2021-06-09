package cache

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const exampleURL = "http://example.com"

type dbRecorder struct {
	getCount int
	mutex    sync.Mutex
}

func (d *dbRecorder) Create(ctx context.Context, id, url string, expiredAt time.Time) error {
	return nil
}

func (d *dbRecorder) Delete(ctx context.Context, id string) error {
	return nil
}

func (d *dbRecorder) Get(ctx context.Context, id string) (string, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.getCount++
	return exampleURL, nil
}

func Test_Cache_Get(t *testing.T) {
	t.Run("same id should only one goroutine can go to database", func(t *testing.T) {
		recorder := dbRecorder{}
		cache := New(&recorder)
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

		assert.Equal(t, 1, recorder.getCount)
	})
	t.Run("diffrent id will go to database only once and hit cache afterwards", func(t *testing.T) {
		t.Skip()

		recorder := dbRecorder{}
		cache := New(&recorder)
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
		assert.Equal(t, numG, recorder.getCount)
		concurrentCall(numG) // second call
		assert.Equal(t, numG, recorder.getCount)
	})
}
