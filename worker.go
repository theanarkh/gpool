package gpool

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/emirpasic/gods/lists/singlylinkedlist"
)

var globalTasks singlylinkedlist.List
var globalTasksMutex sync.Mutex

type worker interface {
	run()
	submit(t Task)
	statistic() *workerInfo
	lastExecTime() time.Time
	stealTasks() []Task
}

var workerId int64 = 0

type Task func()

type taskWorker struct {
	id       int64
	locker   sync.Locker
	cond     *sync.Cond
	tasks    *singlylinkedlist.List
	lastTime atomic.Value
	done     atomic.Int64
	pool     *pool
}

func (w *taskWorker) run() {
	go func() {
		for {
			w.locker.Lock()
			for w.tasks.Size() == 0 {
				if w.pool.flags&ENABLE_STEAL_TASK != 0 {
					w.locker.Unlock()
					tasks := w.pool.stealTasks(w)
					w.locker.Lock()
					if len(tasks) > 0 {
						if w.pool.flags&ENABLE_DEBUG != 0 {
							w.pool.logger.Info("steal wokrers:" + strconv.Itoa(len(tasks)))
						}
						for _, task := range tasks {
							w.tasks.Add(task)
						}
						break
					}
				}
				if w.pool.flags&ENABLE_GLOBAL_TASK_QUEUE != 0 {
					globalTasksMutex.Lock()
					if globalTasks.Size() > 0 {
						max := 5
						for globalTasks.Size() > 0 && max > 0 {
							max--
							value, _ := globalTasks.Get(0)
							globalTasks.Remove(0)
							w.tasks.Add(value)
						}
						if w.pool.flags&ENABLE_DEBUG != 0 {
							w.pool.logger.Info(strconv.FormatInt(w.id, 10) + "get tasks from global tasks queue")
						}

						globalTasksMutex.Unlock()
						break
					}
					globalTasksMutex.Unlock()

				}
				if w.pool.flags&ENABLE_DEBUG != 0 {
					w.pool.logger.Info(strconv.FormatInt(w.id, 10) + "waiting")
				}
				w.cond.Wait()
			}
			value, _ := w.tasks.Get(0)
			w.tasks.Remove(0)
			w.locker.Unlock()
			exited := func() (exited bool) {
				defer func() {
					if err := recover(); err != nil {
						if w.pool.panicHandler != nil {
							w.pool.panicHandler(err)
						}
						w.pool.logger.Info(fmt.Sprintf("panic in worker:  %v: %s", err, debug.Stack()))
					}
				}()
				task := value.(Task)
				if task == nil {
					return true
				}
				task()
				w.done.Add(1)
				w.lastTime.Store(time.Now())
				return false
			}()
			if exited {
				w.pool.logger.Info("worker exit")
				return
			}
		}
	}()
}

func (w *taskWorker) statistic() *workerInfo {
	w.locker.Lock()
	count := w.tasks.Size()
	w.locker.Unlock()
	return &workerInfo{
		Id:         w.id,
		DoneTask:   w.done.Load(),
		UndoneTask: int64(count),
	}
}

func (w *taskWorker) lastExecTime() time.Time {
	w.locker.Lock()
	defer w.locker.Unlock()
	return w.lastTime.Load().(time.Time)
}

func (w *taskWorker) submit(t Task) {
	w.locker.Lock()
	defer w.locker.Unlock()
	// TODO
	if w.pool.flags&ENABLE_GLOBAL_TASK_QUEUE != 0 && w.tasks.Size() >= 5 {
		globalTasksMutex.Lock()
		globalTasks.Add(t)
		globalTasksMutex.Unlock()
	} else {
		w.tasks.Add(t)
		w.cond.Signal()
	}
}

func genWorkerId() int64 {
	return atomic.AddInt64(&workerId, 1)
}

func (w *taskWorker) stealTasks() []Task {
	w.locker.Lock()
	defer w.locker.Unlock()
	count := w.tasks.Size()
	// TODO
	if count >= 3 {
		var tasks []Task
		for i := 0; i < count/3; i++ {
			task, _ := w.tasks.Get(i)
			w.tasks.Remove(i)
			tasks = append(tasks, task.(Task))
		}
		return tasks
	}
	return nil
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
