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

func (suite *cacheTestSuite) SetupTest() {
	suite.dbRecorder = dbRecorder{}
	suite.cache = New(&suite.dbRecorder, zap.NewNop())
	suite.ctx = context.Background()
	suite.numG = 200000
}

func (suite *cacheTestSuite) Test_Get_only_one_goroutine_can_hit_database_if_they_query_same_id() {

	var wg sync.WaitGroup
	wg.Add(suite.numG)
	for i := 0; i < suite.numG; i++ {
		go func() {
			defer wg.Done()
			suite.cache.Get(suite.ctx, "aaaaaa")
		}()
	}
	wg.Wait()

	suite.Equal(1, suite.dbRecorder.getCount)
}

func (suite *cacheTestSuite) Test_Get_every_goroutine_is_able_to_hit_database_once_then_hit_cache_while_next_call() {
	concurrentCall := func(numG int) {
		var wg sync.WaitGroup
		wg.Add(numG)
		for i := 0; i < numG; i++ {
			go func(i int) {
				defer wg.Done()
				suite.cache.Get(suite.ctx, fmt.Sprintln(i))
			}(i)
		}
		wg.Wait()
	}
	concurrentCall(suite.numG) // first call
	suite.Equal(suite.numG, suite.dbRecorder.getCount, "hit database")
	concurrentCall(suite.numG) // second call
	suite.Equal(suite.numG, suite.dbRecorder.getCount, "hit cache, so `getCount` does not increse")
}

func (suite *cacheTestSuite) Test_Delete_should_hit_database() {
	err := suite.cache.Delete(suite.ctx, "aaaaaa")
	suite.NoError(err)
	suite.Equal(1, suite.dbRecorder.deleteCount)
	// TODO: use bloom filter to avoid non-existent record has chance to hit database
}

func Test_cacheTestSuite(t *testing.T) {
	suite.Run(t, new(cacheTestSuite))
}
