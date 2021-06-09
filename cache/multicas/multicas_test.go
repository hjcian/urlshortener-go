package multicas

import (
	"math/rand"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_same_key_should_only_one_set_successfully(t *testing.T) {
	t.Run("no pre-set", func(t *testing.T) {
		// t.Skip()
		c := NewMultiCAS()

		numG := 100000
		ansBucket := make(chan bool, numG)

		var wg sync.WaitGroup
		wg.Add(numG)
		for i := 0; i < numG; i++ {
			go func() {
				defer wg.Done()
				ok := c.Set("key")
				ansBucket <- ok
			}()
		}
		go func() {
			wg.Wait()
			close(ansBucket)
		}()

		sum := 0
		for ans := range ansBucket {
			if ans {
				sum++
			}
		}
		assert.Equal(t, 1, sum)
	})

	t.Run("pre-set", func(t *testing.T) {
		// t.Skip()

		c := NewMultiCAS()
		ok := c.Set("key")
		assert.Equal(t, true, ok)

		numG := 100000
		unsetNum := rand.Intn(numG)
		ansBucket := make(chan bool, numG)

		var wg sync.WaitGroup
		wg.Add(numG)
		for i := 0; i < numG; i++ {
			go func(i int) {
				defer wg.Done()
				if i == unsetNum {
					c.Unset("key")
				} else {
					ok := c.Set("key")
					ansBucket <- ok
				}
			}(i)
		}
		go func() {
			wg.Wait()
			close(ansBucket)
		}()

		sum := 0
		for ans := range ansBucket {
			if ans {
				sum++
			}
		}
		assert.Equal(t, 1, sum)
	})
}

func Test_different_key_should_everyone_set_successfully(t *testing.T) {
	// t.Skip()

	c := NewMultiCAS()

	numG := 100000
	ansBucket := make(chan bool, numG)

	var wg sync.WaitGroup
	wg.Add(numG)
	for i := 0; i < numG; i++ {
		go func(i int) {
			defer wg.Done()
			ok := c.Set(i)
			ansBucket <- ok
		}(i)
	}
	go func() {
		wg.Wait()
		close(ansBucket)
	}()

	sum := 0
	for ans := range ansBucket {
		if ans {
			sum++
		}
	}
	assert.Equal(t, numG, sum)
}
