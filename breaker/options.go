package breaker

import (
	"time"
)

var defaultOption = options{
	maxRequests: 10,
	interval:    60 * time.Second,
	timeout:     10 * time.Second,
	readyToTrip: func(c Counts) bool {
		failureRatio := float64(c.TotalFailures) / float64(c.Requests)
		return c.Requests >= 60 && failureRatio >= 0.6
	},
}

type options struct {
	maxRequests uint32
	interval    time.Duration
	timeout     time.Duration
	readyToTrip func(Counts) bool
}

type Option func(*options)

func WithMaxRequests(r uint32) Option {
	return func(o *options) {
		o.maxRequests = r
	}
}

func WithInterval(t time.Duration) Option {
	return func(o *options) {
		o.interval = t
	}
}

func WithTimeout(t time.Duration) Option {
	return func(o *options) {
		o.timeout = t
	}
}

func WithReadyToTrip(f func(Counts) bool) Option {
	return func(o *options) {
		o.readyToTrip = f
	}
}
