package naming

import "time"

var defaultRegistryOption = registryOptions{
	timeout: 2 * time.Second,
	ttl:     3,
}

type registryOptions struct {
	timeout time.Duration
	ttl     int64
}

type RegistryOption func(*registryOptions)

func WithTimeout(t time.Duration) RegistryOption {
	return func(o *registryOptions) {
		o.timeout = t
	}
}

func WithTTL(t int64) RegistryOption {
	return func(o *registryOptions) {
		o.ttl = t
	}
}
