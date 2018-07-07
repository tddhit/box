package breaker

import (
	"github.com/sony/gobreaker"
)

type Counts gobreaker.Counts

type Breaker struct {
	opt options
	*gobreaker.CircuitBreaker
}

func New(opts ...Option) *Breaker {
	opt := defaultOption
	for _, o := range opts {
		o(&opt)
	}
	st := gobreaker.Settings{
		Interval:    opt.interval,
		Timeout:     opt.timeout,
		MaxRequests: opt.maxRequests,
		ReadyToTrip: func(c gobreaker.Counts) bool {
			return opt.readyToTrip(Counts(c))
		},
	}
	cb := gobreaker.NewCircuitBreaker(st)
	return &Breaker{
		opt:            opt,
		CircuitBreaker: cb,
	}
}
