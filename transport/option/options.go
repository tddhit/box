package option

import (
	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/tddhit/box/interceptor"
	"github.com/tddhit/box/naming"
)

type ServerOptions struct {
	RegistryKey       string
	Registry          *naming.Registry
	GatewayMux        *runtime.ServeMux
	UnaryMiddlewares  []interceptor.UnaryServerMiddleware
	StreamMiddlewares []interceptor.StreamServerMiddleware
	FuncBeforeClose   func()
	FuncAfterClose    func()
}

type ServerOption func(*ServerOptions)

func WithUnaryServerMiddleware(ms ...interceptor.UnaryServerMiddleware) ServerOption {
	return func(o *ServerOptions) {
		o.UnaryMiddlewares = append(o.UnaryMiddlewares, ms...)
	}
}

func WithStreamServerMiddleware(
	ms ...interceptor.StreamServerMiddleware) ServerOption {

	return func(o *ServerOptions) {
		o.StreamMiddlewares = append(o.StreamMiddlewares, ms...)
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

func WithBeforeClose(f func()) ServerOption {
	return func(o *ServerOptions) {
		o.FuncBeforeClose = f
	}
}

func WithAfterClose(f func()) ServerOption {
	return func(o *ServerOptions) {
		o.FuncAfterClose = f
	}
}

type DialOptions struct {
	Balancer          string
	UnaryMiddlewares  []interceptor.UnaryClientMiddleware
	StreamMiddlewares []interceptor.StreamClientMiddleware
}

type DialOption func(*DialOptions)

func WithUnaryClientMiddleware(m interceptor.UnaryClientMiddleware) DialOption {
	return func(o *DialOptions) {
		o.UnaryMiddlewares = append(o.UnaryMiddlewares, m)
	}
}

func WithStreamClientMiddleware(m interceptor.StreamClientMiddleware) DialOption {
	return func(o *DialOptions) {
		o.StreamMiddlewares = append(o.StreamMiddlewares, m)
	}
}

func WithBalancer(b string) DialOption {
	return func(o *DialOptions) {
		o.Balancer = b
	}
}

type CallOptions struct {
}

type CallOption func(*CallOptions)
