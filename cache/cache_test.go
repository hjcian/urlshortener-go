package cache

import (
	"context"
	"fmt"
	"goshorturl/repository"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"
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

type cacheTestSuite struct {
	suite.Suite
	dbRecorder dbRecorder
	cache      repository.Repository
	ctx        context.Context
	numG       int
}

func (suit *cacheTestSuite) SetupTest() {
	suit.dbRecorder = dbRecorder{}
	suit.cache = New(&suit.dbRecorder, zap.NewNop())
	suit.ctx = context.Background()
	suit.numG = 200000
}

func (suit *cacheTestSuite) Test_Get_only_one_goroutine_can_hit_database_if_they_query_same_id() {

	var wg sync.WaitGroup
	wg.Add(suit.numG)
	for i := 0; i < suit.numG; i++ {
		go func() {
			defer wg.Done()
			suit.cache.Get(suit.ctx, "aaaaaa")
		}()
	}
	wg.Wait()

	suit.Equal(1, suit.dbRecorder.getCount)
}

func (suit *cacheTestSuite) Test_Get_every_goroutine_is_able_to_hit_database_once_then_hit_cache_while_next_call() {
	concurrentCall := func(numG int) {
		var wg sync.WaitGroup
		wg.Add(numG)
		for i := 0; i < numG; i++ {
			go func(i int) {
				defer wg.Done()
				suit.cache.Get(suit.ctx, fmt.Sprintln(i))
			}(i)
		}
		wg.Wait()
	}
	concurrentCall(suit.numG) // first call
	suit.Equal(suit.numG, suit.dbRecorder.getCount, "hit database")
	concurrentCall(suit.numG) // second call
	suit.Equal(suit.numG, suit.dbRecorder.getCount, "hit cache, so `getCount` does not increse")
}

func (suit *cacheTestSuite) Test_Delete_should_hit_database() {
	err := suit.cache.Delete(suit.ctx, "aaaaaa")
	suit.NoError(err)
	suit.Equal(1, suit.dbRecorder.deleteCount)
	// TODO: use bloom filter to avoid non-existent record has chance to hit database
}

func Test_cacheTestSuite(t *testing.T) {
	suite.Run(t, new(cacheTestSuite))
}
