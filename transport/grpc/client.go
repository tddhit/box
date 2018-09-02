package grpc

import (
	"context"

	"google.golang.org/grpc"
	_ "google.golang.org/grpc/balancer/roundrobin"

	"github.com/tddhit/box/interceptor"
	_ "github.com/tddhit/box/resolver/etcd"
	"github.com/tddhit/box/transport/common"
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
		grpc.WithStreamInterceptor(c.streamInterceptor),
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
	h := interceptor.ChainUnaryClientMiddleware(f, c.opts.UnaryMiddlewares...)
	return h(ctx, method, req, reply)

}

func (c *GRPCClient) streamInterceptor(ctx context.Context,
	desc *grpc.StreamDesc, cc *grpc.ClientConn, method string,
	streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {

	f := func(ctx context.Context, desc *common.StreamDesc,
		method string) (common.ClientStream, error) {

		return streamer(ctx, (*grpc.StreamDesc)(desc), cc, method, opts...)
	}
	h := interceptor.ChainStreamClientMiddleware(f, c.opts.StreamMiddlewares...)
	return h(ctx, (*common.StreamDesc)(desc), method)
}

func (c *GRPCClient) Invoke(ctx context.Context, method string,
	args interface{}, reply interface{}, opts ...option.CallOption) error {

	return c.ClientConn.Invoke(ctx, method, args, reply)
}

func (c *GRPCClient) Close() {
	c.ClientConn.Close()
}

func (c *GRPCClient) NewStream(ctx context.Context, desc common.ServiceDesc, i int,
	method string, opts ...option.CallOption) (common.ClientStream, error) {

	sd := desc.Desc().(*grpc.ServiceDesc)
	streamDesc := &sd.Streams[i]
	return c.ClientConn.NewStream(ctx, streamDesc, method)
}
