package multicas

import (
	"math/rand"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"
)

type multiCASVerSuite struct {
	name     string
	multiCAS MultiCAS
}

type multiCASTestSuite struct {
	suite.Suite
	numG     int
	versions []multiCASVerSuite
	wg       *sync.WaitGroup
}

func (suite *multiCASTestSuite) SetupTest() {
	suite.versions = []multiCASVerSuite{
		{
			"version 1 - use sync.Mutex",
			newMultiCAS_v1_forTest(),
		},
		{
			"version 2 - use sync.RWMutex",
			newMultiCAS_v2_forTest(),
		},
	}
	suite.numG = 300000
	suite.wg = new(sync.WaitGroup)
}

func (suite *multiCASTestSuite) Test_same_key_should_only_one_set_successfully_no_preset() {
	for _, ver := range suite.versions {
		suite.Run(ver.name, func() {
			ansBucket := make(chan bool, suite.numG)

			suite.wg.Add(suite.numG)
			for i := 0; i < suite.numG; i++ {
				go func() {
					defer suite.wg.Done()
					ok := ver.multiCAS.Set("key")
					ansBucket <- ok
				}()
			}
			go func() {
				suite.wg.Wait()
				close(ansBucket)
			}()

			sum := 0
			for ans := range ansBucket {
				if ans {
					sum++
				}
			}
			suite.Equal(1, sum)
		})
	}
}

func (suite *multiCASTestSuite) Test_same_key_should_only_one_set_successfully_preset() {
	for _, ver := range suite.versions {
		suite.Run(ver.name, func() {
			ok := ver.multiCAS.Set("key")
			suite.Equal(true, ok)

			unsetNum := rand.Intn(suite.numG)
			ansBucket := make(chan bool, suite.numG)

			suite.wg.Add(suite.numG)
			for i := 0; i < suite.numG; i++ {
				go func(i int) {
					defer suite.wg.Done()
					if i == unsetNum {
						ver.multiCAS.Unset("key")
					} else {
						ok := ver.multiCAS.Set("key")
						ansBucket <- ok
					}
				}(i)
			}
			go func() {
				suite.wg.Wait()
				close(ansBucket)
			}()

			sum := 0
			for ans := range ansBucket {
				if ans {
					sum++
				}
			}
			suite.Equal(1, sum)
		})
	}
}

func (suite *multiCASTestSuite) Test_different_key_every_GR_set_successfully() {
	for _, ver := range suite.versions {
		suite.Run(ver.name, func() {
			ansBucket := make(chan bool, suite.numG)
			suite.wg.Add(suite.numG)
			for i := 0; i < suite.numG; i++ {
				go func(i int) {
					defer suite.wg.Done()
					ok := ver.multiCAS.Set(i)
					ansBucket <- ok
				}(i)
			}
			go func() {
				suite.wg.Wait()
				close(ansBucket)
			}()

			sum := 0
			for ans := range ansBucket {
				if ans {
					sum++
				}
			}
			suite.Equal(suite.numG, sum)
		})
	}
}

func Test_suiteTestMultiCAS(t *testing.T) {
	suite.Run(t, new(multiCASTestSuite))
}
