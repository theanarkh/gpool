package gpool

import "time"

type Option func(*pool)

func WithCapacity(capacity int) Option {
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

func WithPanicHandler(handler PanicHandler) Option {
	return func(p *pool) {
		p.panicHandler = handler
	}
}

func WithFlags(flags int) Option {
	return func(p *pool) {
		p.flags = flags
	}
}
