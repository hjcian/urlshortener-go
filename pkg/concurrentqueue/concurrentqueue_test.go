package concurrentqueue

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Enqueue(t *testing.T) {
	queue := New()
	numG := 100000
	var wg sync.WaitGroup
	wg.Add(numG)
	for i := 0; i < numG; i++ {
		go func(i int) {
			defer wg.Done()
			queue.Enqueue(fmt.Sprintln(i))
		}(i)
	}
	wg.Wait()
	assert.Equal(t, numG, queue.Len())
}

func Test_BatchEnqueue(t *testing.T) {
	queue := New()
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
			queue.BatchEnqueue(ids)
		}(i)
	}
	wg.Wait()
	assert.Equal(t, numG*factor, queue.Len())
}

func Test_Dequeue(t *testing.T) {
	t.Run("empty should return err", func(t *testing.T) {
		queue := New()
		ele, err := queue.Dequeue()
		assert.Error(t, err)
		assert.Empty(t, ele)
	})
	t.Run("empty should return err", func(t *testing.T) {
		queue := New()
		numG := 100000
		for i := 0; i < numG; i++ {
			queue.Enqueue(fmt.Sprintln(i))
		}

		numDecrease := numG / 2
		errCollect := make(chan error, numDecrease)
		var wg sync.WaitGroup
		wg.Add(numDecrease)
		for i := 0; i < numDecrease; i++ {
			go func(i int) {
				defer wg.Done()
				_, err := queue.Dequeue()
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
		assert.Equal(t, numG-numDecrease, queue.Len())
	})

}
