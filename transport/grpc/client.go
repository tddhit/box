package grpc

import (
	"context"

	"google.golang.org/grpc"
	_ "google.golang.org/grpc/balancer/roundrobin"

	_ "github.com/tddhit/box/resolver/etcd"
	"github.com/tddhit/box/transport/option"
)

type GRPCClient struct {
	*grpc.ClientConn
	opt option.DialOptions
}

func DialContext(ctx context.Context, target string,
	opts ...option.DialOption) (*GRPCClient, error) {

	var (
		opt  option.DialOptions
		conn *grpc.ClientConn
		err  error
	)
	for _, o := range opts {
		o(&opt)
	}
	c := &GRPCClient{
		opt: opt,
	}
	if c.opt.Balancer != "" {
		conn, err = grpc.DialContext(ctx, target,
			grpc.WithInsecure(), grpc.WithBalancerName(c.opt.Balancer))
	} else {
		conn, err = grpc.DialContext(ctx, target, grpc.WithInsecure())
	}
	if err != nil {
		return nil, err
	}
	c.ClientConn = conn
	return c, nil
}

func (c *GRPCClient) Invoke(ctx context.Context, method string,
	args interface{}, reply interface{}, opts ...option.CallOption) error {

	return grpc.Invoke(ctx, method, args, reply, c.ClientConn)
}

func (c *GRPCClient) Close() {
	c.ClientConn.Close()
}
