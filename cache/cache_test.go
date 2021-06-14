package cache

import (
	"context"
	"errors"
	"fmt"
	"goshorturl/repository"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

const (
	exampleID  = "aaaaaa"
	exampleURL = "http://example.com"
)

var (
	errStorageInternalError = errors.New("storage internal error")
)

type dbRecorder struct {
	repository.UnimplementedRepository
	errorMode   bool
	mutex       sync.Mutex
	getCount    int
	deleteCount int
	createCount int
	updateCount int
}

func (d *dbRecorder) Get(ctx context.Context, id string) (string, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.errorMode {
		return "", errStorageInternalError
	}
	d.getCount++
	return exampleURL, nil
}
func (d *dbRecorder) Delete(ctx context.Context, id string) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.errorMode {
		return errStorageInternalError
	}
	d.deleteCount++
	return nil
}

func (d *dbRecorder) Create(ctx context.Context, id, url string, expiredAt time.Time) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.errorMode {
		return errStorageInternalError
	}
	d.createCount++
	return nil
}

func (d *dbRecorder) Update(ctx context.Context, id, url string, expiredAt time.Time) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.errorMode {
		return errStorageInternalError
	}
	d.updateCount++
	return nil
}

// enableError enables the error mode that will cause every operation to return errStorageInternalError.
func (d *dbRecorder) enableError() {
	d.errorMode = true
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
	suite.cache = New(&suite.dbRecorder, zap.NewNop(), UseInMemoryCache())
	suite.ctx = context.Background()
	suite.numG = 200000
}

func (suite *cacheTestSuite) Test_Get_only_one_goroutine_can_hit_database_if_they_query_same_id() {
	var wg sync.WaitGroup
	wg.Add(suite.numG)
	for i := 0; i < suite.numG; i++ {
		go func() {
			defer wg.Done()
			suite.cache.Get(suite.ctx, exampleID)
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

func (suite *cacheTestSuite) Test_Delete_hit_database() {
	// NOTE: this test case will turn out to be invalid after using bloom filter in cache
	err := suite.cache.Delete(suite.ctx, exampleID)
	suite.NoError(err)
	suite.Equal(1, suite.dbRecorder.deleteCount)
}

func (suite *cacheTestSuite) Test_Create_cache_the_entry() {
	err := suite.cache.Create(suite.ctx, exampleID, exampleURL, time.Now().Add(24*time.Hour))
	suite.NoError(err)
	suite.Equal(1, suite.dbRecorder.createCount, "should create OK")

	got, err := suite.cache.Get(suite.ctx, exampleID)
	suite.Equal(exampleURL, got, "should retrieve the original URL")
	suite.NoError(err)
	suite.Equal(0, suite.dbRecorder.getCount, "should retrieve from cache instead storage")
}

func (suite *cacheTestSuite) Test_Create_cache_the_entry_fail() {
	suite.dbRecorder.enableError()

	err := suite.cache.Create(suite.ctx, exampleID, exampleURL, time.Now().Add(24*time.Hour))
	suite.Equal(errStorageInternalError, err)
	suite.Equal(0, suite.dbRecorder.createCount, "should not create anything")

	got, err := suite.cache.Get(suite.ctx, exampleID)
	suite.Equal("", got, "should got nothing")
	suite.Equal(errStorageInternalError, err)
	suite.Equal(0, suite.dbRecorder.getCount, "should not retrieve anything")
}

func (suite *cacheTestSuite) Test_Update_cache_the_entry() {
	err := suite.cache.Update(suite.ctx, exampleID, exampleURL, time.Now().Add(24*time.Hour))
	suite.NoError(err)
	suite.Equal(1, suite.dbRecorder.updateCount, "should update OK")

	got, err := suite.cache.Get(suite.ctx, exampleID)
	suite.Equal(exampleURL, got, "should retrieve the original URL")
	suite.NoError(err)
	suite.Equal(0, suite.dbRecorder.getCount, "should retrieve from cache instead storage")
}

func (suite *cacheTestSuite) Test_Update_cache_the_entry_fail() {
	suite.dbRecorder.enableError()

	err := suite.cache.Update(suite.ctx, exampleID, exampleURL, time.Now().Add(24*time.Hour))
	suite.Equal(errStorageInternalError, err)
	suite.Equal(0, suite.dbRecorder.updateCount, "should not update anything")

	got, err := suite.cache.Get(suite.ctx, exampleID)
	suite.Equal("", got, "should got nothing")
	suite.Equal(errStorageInternalError, err)
	suite.Equal(0, suite.dbRecorder.getCount, "should not retrieve anything")
}

func Test_cacheTestSuite(t *testing.T) {
	suite.Run(t, new(cacheTestSuite))
}
