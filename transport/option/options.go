package option

import (
	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/tddhit/box/interceptor"
	"github.com/tddhit/box/naming"
)

type ServerOptions struct {
	RegistryKey string
	Registry    *naming.Registry
	GatewayMux  *runtime.ServeMux
	Middlewares []interceptor.Middleware
}

type ServerOption func(*ServerOptions)

func WithMiddleware(m interceptor.Middleware) ServerOption {
	return func(o *ServerOptions) {
		o.Middlewares = append(o.Middlewares, m)
	}
}

func WithRegistry(r *naming.Registry, k string) ServerOption {
	return func(o *ServerOptions) {
		o.Registry = r
		o.RegistryKey = k
	}
}

func WithGatewayMux(m *runtime.ServeMux) ServerOption {
	return func(o *ServerOptions) {
		o.GatewayMux = m
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
