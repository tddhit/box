package grpc

import (
	"context"
	"net"

	"google.golang.org/grpc"

	"github.com/tddhit/box/interceptor"
	"github.com/tddhit/box/transport/common"
	"github.com/tddhit/box/transport/option"
)

type GrpcTransport struct {
	*grpc.Server
	opts    option.ServerOptions
	lis     net.Listener
	handler interceptor.UnaryHandler
}

func New(lis net.Listener,
	opts ...option.ServerOption) *GrpcTransport {

	var ops option.ServerOptions
	for _, o := range opts {
		o(&ops)
	}
	s := &GrpcTransport{
		opts: ops,
		lis:  lis,
	}
	s.Server = grpc.NewServer(
		grpc.UnaryInterceptor(s.unaryInterceptor),
		grpc.StreamInterceptor(s.streamInterceptor),
	)
	return s
}

func (s *GrpcTransport) unaryInterceptor(ctx context.Context,
	req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (interface{}, error) {

	f := func(ctx context.Context, req interface{},
		info *common.UnaryServerInfo) (interface{}, error) {

		return handler(ctx, req)
	}

	h := interceptor.ChainUnaryServerMiddleware(f, s.opts.UnaryMiddlewares...)
	return h(ctx, req, (*common.UnaryServerInfo)(info))
}

func (s *GrpcTransport) streamInterceptor(srv interface{}, ss grpc.ServerStream,
	info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

	f := func(srv interface{}, ss common.ServerStream,
		info *common.StreamServerInfo) error {

		return handler(srv, ss)
	}
	h := interceptor.ChainStreamServerMiddleware(f, s.opts.StreamMiddlewares...)
	return h(srv, ss, (*common.StreamServerInfo)(info))
}

func (s *GrpcTransport) Register(desc common.ServiceDesc,
	service interface{}) {

	s.Server.RegisterService(desc.Desc().(*grpc.ServiceDesc), service)
}

func (s *GrpcTransport) Close() {
	if s.opts.FuncBeforeClose != nil {
		s.opts.FuncBeforeClose()
	}
	s.Server.GracefulStop()
	if s.opts.FuncAfterClose != nil {
		s.opts.FuncAfterClose()
	}
}
