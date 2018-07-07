package option

import "github.com/tddhit/box/naming"

type ServerOptions struct {
	Registry    *naming.Registry
	RegistryKey string
}

type ServerOption func(*ServerOptions)

func WithRegistry(r *naming.Registry, k string) ServerOption {
	return func(o *ServerOptions) {
		o.Registry = r
		o.RegistryKey = k
	}
}

type DialOptions struct {
	Balancer string
}

type DialOption func(*DialOptions)

func WithBalancer(b string) DialOption {
	return func(o *DialOptions) {
		o.Balancer = b
	}
}

type CallOptions struct {
}

type CallOption func(*CallOptions)
