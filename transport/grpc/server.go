package grpc

import (
	"net"

	"google.golang.org/grpc"

	"github.com/tddhit/box/transport/common"
	"github.com/tddhit/box/transport/option"
)

type GrpcTransport struct {
	*grpc.Server
	opts option.ServerOptions
	lis  net.Listener
}

func New(lis net.Listener,
	opts ...option.ServerOption) *GrpcTransport {

	var ops option.ServerOptions
	for _, o := range opts {
		o(&ops)
	}
	s := &GrpcTransport{
		Server: grpc.NewServer(),
		opts:   ops,
		lis:    lis,
	}
	return s
}

func (s *GrpcTransport) Register(desc common.ServiceDesc,
	service interface{}) {

	s.Server.RegisterService(desc.Desc().(*grpc.ServiceDesc), service)
}

func (s *GrpcTransport) Close() {
	s.Server.GracefulStop()
}
