package gpool

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPool(t *testing.T) {
	pool, _ := NewPool(WithCapacity(10))
	var count int32 = 0
	var w sync.WaitGroup
	for i := 0; i < 100; i++ {
		w.Add(1)
		pool.Submit(func() {
			atomic.AddInt32(&count, 1)
			w.Done()
		})
	}
	w.Wait()
	result, _ := json.MarshalIndent(pool.Statistic(), "", "    ")
	fmt.Println(string(result))
	assert.Equal(t, int32(100), count, "test error")
}
