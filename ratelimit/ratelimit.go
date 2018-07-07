package ratelimit

import "golang.org/x/time/rate"

type Limiter struct {
	*rate.Limiter
	opt options
}

func New(opts ...Option) *Limiter {
	opt := defaultOption
	for _, o := range opts {
		o(&opt)
	}
	return &Limiter{
		Limiter: rate.NewLimiter(opt.limit, opt.burst),
		opt:     opt,
	}
}
