package gpool

import "time"

type Option func(*pool)

func WithCapacity(capacity int64) Option {
	return func(p *pool) {
		p.cap = capacity
	}
}

func WithMaxIdleTime(maxIdleTime time.Duration) Option {
	return func(p *pool) {
		p.maxIdleTime = maxIdleTime
	}
}

func WithPollIdleWorkerInterval(interval time.Duration) Option {
	return func(p *pool) {
		p.pollIdleWorkerInterval = interval
	}
}

func WithLogger(logger Logger) Option {
	return func(p *pool) {
		p.logger = logger
	}
}

func WithOnPanic(handler PanicHandler) Option {
	return func(p *pool) {
		p.onPanic = handler
	}
}
