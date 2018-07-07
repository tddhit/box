package ratelimit

import (
	"math"

	"golang.org/x/time/rate"
)

var defaultOption = options{
	limit: math.MaxFloat64,
	burst: math.MaxInt64,
}

type options struct {
	limit rate.Limit
	burst int
}

type Option func(*options)

func WithLimit(l float64) Option {
	return func(o *options) {
		o.limit = rate.Limit(l)
	}
}

func WithBurst(b int) Option {
	return func(o *options) {
		o.burst = b
	}
}
