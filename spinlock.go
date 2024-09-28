package gpool

import (
	"runtime"
	"sync/atomic"
)

type spinLock struct {
	lock uint32
}

func (s *spinLock) Lock() {
	var attempt uint32 = 0
	for !atomic.CompareAndSwapUint32(&s.lock, 0, 1) {
		backoff := 1 << attempt
		if backoff > 10 {
			backoff = 10
		}
		for i := 0; i < backoff; i++ {
			runtime.Gosched()
		}
		attempt++
	}
}

func (s *spinLock) Unlock() {
	atomic.StoreUint32(&s.lock, 0)
}

func newSpinLock() *spinLock {
	return &spinLock{}
}
