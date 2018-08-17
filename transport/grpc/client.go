package grpc

import (
	"context"

	"google.golang.org/grpc"
	_ "google.golang.org/grpc/balancer/roundrobin"

	"github.com/tddhit/box/interceptor"
	_ "github.com/tddhit/box/resolver/etcd"
	"github.com/tddhit/box/transport/option"
)

type GRPCClient struct {
	*grpc.ClientConn
	opts option.DialOptions
}

func DialContext(ctx context.Context, target string,
	opts ...option.DialOption) (*GRPCClient, error) {

	var (
		ops  option.DialOptions
		conn *grpc.ClientConn
		err  error
	)
	for _, o := range opts {
		o(&ops)
	}
	c := &GRPCClient{
		opts: ops,
	}
	var grpcOpts = []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(c.unaryInterceptor),
	}
	if c.opts.Balancer != "" {
		grpcOpts = append(grpcOpts, grpc.WithBalancerName(c.opts.Balancer))
	}
	conn, err = grpc.DialContext(ctx, target, grpcOpts...)
	if err != nil {
		return nil, err
	}
	c.ClientConn = conn
	return c, nil
}

func (c *GRPCClient) unaryInterceptor(ctx context.Context,
	method string, req, reply interface{}, cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

	f := func(ctx context.Context, method string, req, reply interface{}) error {
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	h := interceptor.ChainUnaryClient(f, c.opts.Middlewares...)
	return h(ctx, method, req, reply)

}

func (c *GRPCClient) Invoke(ctx context.Context, method string,
	args interface{}, reply interface{}, opts ...option.CallOption) error {

	return grpc.Invoke(ctx, method, args, reply, c.ClientConn)
}

func (c *GRPCClient) Close() {
	c.ClientConn.Close()
}
