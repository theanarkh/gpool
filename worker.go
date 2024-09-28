package gpool

import (
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/emirpasic/gods/lists/singlylinkedlist"
)

type worker interface {
	run()
	submit(t Task)
	statistic() *workerInfo
	lastExecTime() time.Time
}

var workerId int64 = 0

type Task func()

type taskWorker struct {
	id       int64
	locker   sync.Locker
	cond     *sync.Cond
	tasks    *singlylinkedlist.List
	lastTime time.Time
	done     int64
	pool     *pool
}

func (w *taskWorker) run() {
	go func() {
		for {
			w.locker.Lock()
			for w.tasks.Size() == 0 {
				w.cond.Wait()
			}
			value, _ := w.tasks.Get(0)
			w.tasks.Remove(0)
			w.done++
			w.locker.Unlock()
			exited := func() (exited bool) {
				defer func() {
					if r := recover(); r != nil {
						w.pool.logger.Info(fmt.Sprintf("panic in worker:  %v: %s", r, debug.Stack()))
					}
				}()
				task := value.(Task)
				if task == nil {
					return true
				}
				task()
				return false
			}()
			w.locker.Lock()
			w.lastTime = time.Now()
			w.locker.Unlock()
			if exited {
				w.pool.logger.Info("worker exit")
				return
			}
		}
	}()
}

func (w *taskWorker) statistic() *workerInfo {
	w.locker.Lock()
	defer w.locker.Unlock()
	return &workerInfo{
		Id:         w.id,
		DoneTask:   w.done,
		UndoneTask: int64(w.tasks.Size()),
	}
}

func (w *taskWorker) lastExecTime() time.Time {
	w.locker.Lock()
	defer w.locker.Unlock()
	return w.lastTime
}

func (w *taskWorker) submit(t Task) {
	w.locker.Lock()
	defer w.locker.Unlock()
	w.tasks.Add(t)
	w.cond.Signal()
}

func genWorkerId() int64 {
	return atomic.AddInt64(&workerId, 1)
}

func newWorker(p *pool) worker {
	id := genWorkerId()
	w := &taskWorker{
		pool:   p,
		locker: newSpinLock(),
		tasks:  singlylinkedlist.New(),
		id:     id,
	}
	w.cond = sync.NewCond(w.locker)
	return w
}
