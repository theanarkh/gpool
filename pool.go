package gpool

import (
	"errors"
	"sync"
	"time"
)

type workerInfo struct {
	Id         int64 `json:"id"`
	DoneTask   int64 `json:"done_task"`
	UndoneTask int64 `json:"undone_task"`
}

type StatisticInfo struct {
	Workers []*workerInfo
}

type Pool interface {
	Submit(t Task) error
	Statistic() *StatisticInfo
}

type PanicHandler func(string)

type pool struct {
	locker                 sync.Locker
	cap                    int64
	count                  int64
	maxIdleTime            time.Duration
	pollIdleWorkerInterval time.Duration
	logger                 Logger
	onPanic                PanicHandler
	workers                []worker
	once                   sync.Once
	next                   int64
}

func (p *pool) Submit(t Task) error {
	if t == nil {
		return errors.New("invalid task")
	}
	worker := p.selectWorker()
	worker.submit(t)
	return nil
}

func (p *pool) Statistic() *StatisticInfo {
	p.locker.Lock()
	defer p.locker.Unlock()
	statistic := &StatisticInfo{}
	for _, worker := range p.workers {
		statistic.Workers = append(statistic.Workers, worker.statistic())
	}
	return statistic
}

func (p *pool) poll() {
	t := time.NewTicker(p.pollIdleWorkerInterval)
	for {
		<-t.C
		p.locker.Lock()
		var expired bool
		var workers []worker
		for _, worker := range p.workers {
			if time.Since(worker.lastExecTime()) > p.maxIdleTime {
				expired = true
				p.count--
				worker.submit(nil)
			} else {
				workers = append(workers, worker)
			}
		}
		if expired {
			p.workers = workers
		}
		p.locker.Unlock()
	}
}

func (p *pool) selectWorker() worker {
	p.locker.Lock()
	defer p.locker.Unlock()
	if p.count < p.cap {
		p.count++
		worker := newWorker(p)
		worker.run()
		p.once.Do(func() {
			go p.poll()
		})
		p.workers = append(p.workers, worker)
		return worker
	}
	next := p.next
	p.next = (p.next + 1) % int64(len(p.workers))
	return p.workers[next]
}

func NewPool(options ...Option) (Pool, error) {
	p := &pool{
		cap:                    10,
		maxIdleTime:            time.Second * 10,
		pollIdleWorkerInterval: time.Second * 30,
		logger:                 newLogger(),
		locker:                 newSpinLock(),
	}
	for _, option := range options {
		option(p)
	}
	return p, nil
}
