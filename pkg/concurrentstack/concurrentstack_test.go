package concurrentstack

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Push(t *testing.T) {
	stack := New()
	numG := 100000
	var wg sync.WaitGroup
	wg.Add(numG)
	for i := 0; i < numG; i++ {
		go func(i int) {
			defer wg.Done()
			stack.Push(fmt.Sprintln(i))
		}(i)
	}
	wg.Wait()
	assert.Equal(t, numG, stack.Len())
}

func Test_BatchPush(t *testing.T) {
	stack := New()
	factor := 2
	numG := 100000
	var wg sync.WaitGroup
	wg.Add(numG)
	for i := 0; i < numG; i++ {
		go func(i int) {
			defer wg.Done()
			ids := make([]string, 0, factor)
			for j := 0; j < factor; j++ {
				ids = append(ids, fmt.Sprintln(j))
			}
			stack.BatchPush(ids)
		}(i)
	}
	wg.Wait()
	assert.Equal(t, numG*factor, stack.Len())
}

func Test_Pop(t *testing.T) {
	t.Run("empty should return err", func(t *testing.T) {
		stack := New()
		ele, err := stack.Pop()
		assert.Error(t, err)
		assert.Empty(t, ele)
	})
	t.Run("empty should return err", func(t *testing.T) {
		stack := New()
		numG := 100000
		for i := 0; i < numG; i++ {
			stack.Push(fmt.Sprintln(i))
		}

		numDecrease := numG / 2
		errCollect := make(chan error, numDecrease)
		var wg sync.WaitGroup
		wg.Add(numDecrease)
		for i := 0; i < numDecrease; i++ {
			go func(i int) {
				defer wg.Done()
				_, err := stack.Pop()
				errCollect <- err
			}(i)
		}
		wg.Wait()
		close(errCollect)

		countErr := 0
		for err := range errCollect {
			if err != nil {
				countErr++
			}
		}
		assert.Equal(t, 0, countErr)
		assert.Equal(t, numG-numDecrease, stack.Len())
	})

}
