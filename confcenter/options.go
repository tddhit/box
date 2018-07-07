package confcenter

import (
	"time"
)

var defaultOption = options{
	timeout: 2 * time.Second,
}

type options struct {
	timeout  time.Duration
	savePath string
}

type Option func(*options)

func WithTimeout(t time.Duration) Option {
	return func(o *options) {
		o.timeout = t
	}
}

func WithSavePath(p string) Option {
	return func(o *options) {
		o.savePath = p
	}
}
