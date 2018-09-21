package transport

import (
	"context"
	"errors"
	"net"
	"os"
	"strings"
	"time"

	mwcommon "github.com/tddhit/box/mw/common"
	"github.com/tddhit/box/socket"
	trcommon "github.com/tddhit/box/transport/common"
	grpctr "github.com/tddhit/box/transport/grpc"
	httptr "github.com/tddhit/box/transport/http"
	"github.com/tddhit/box/transport/option"
	"github.com/tddhit/box/util"
	"github.com/tddhit/tools/log"
)

var (
	errInvalidListenTarget = errors.New(`
Invalid listen target. e.g. [grpc | http]://[127.0.0.1]:8090")`)

	errInvalidDialTarget = errors.New(`
Invalid dial target. e.g. 	 
	[grpc | http]://127.0.0.1:8090
	etcd://127.0.0.1:2379/echoservice`)
)

type Transport interface {
	Register(desc trcommon.ServiceDesc, ss interface{})
	Serve(net.Listener) error
	Close()
}

type Server struct {
	Transport
	opts   option.ServerOptions
	addr   string
	lis    net.Listener
	cancel context.CancelFunc
	startC chan struct{}
}

func (s *Server) Addr() string {
	return s.addr
}

func (s *Server) RegisterAddr() {
	if s.opts.Registry != nil {
		s.cancel = s.opts.Registry.Register(s.opts.RegistryKey, s.addr)
	}
}

func (s *Server) UnregisterAddr() {
	if s.opts.Registry != nil {
		log.Info("unregister")
		s.cancel()
		time.Sleep(time.Duration(s.opts.Registry.TTL())*time.Second +
			time.Second)
	}
}

func (s *Server) Serve() error {
	close(s.startC)
	err := s.Transport.Serve(s.lis)
	log.Warn(s.addr, "Server Close:", err)
	return err
}

func (s *Server) Started() <-chan struct{} {
	return s.startC
}

type ClientConn interface {
	Invoke(ctx context.Context, method string, args interface{},
		reply interface{}, opts ...option.CallOption) error
	NewStream(ctx context.Context, desc trcommon.ServiceDesc, i int,
		method string, opts ...option.CallOption) (trcommon.ClientStream, error)
	Close()
}

func Listen(target string, opts ...option.ServerOption) (*Server, error) {
	var ops option.ServerOptions
	for _, o := range opts {
		o(&ops)
	}

	// parse proto/addr
	s := strings.Split(target, "://")
	if len(s) != 2 {
		return nil, errInvalidListenTarget
	}
	proto, addr := s[0], s[1]
	s = strings.Split(addr, ":")
	if len(s) != 2 {
		return nil, errInvalidListenTarget
	}
	ip := s[0]
	if ip == "" {
		addr = util.GetLocalAddr(addr)
	}

	server := &Server{
		opts:   ops,
		addr:   addr,
		startC: make(chan struct{}),
	}
	if os.Getenv(mwcommon.FORK) == "1" {
		lis, err := socket.Listen(addr)
		if err != nil {
			return nil, err
		}
		server.lis = lis
	}
	switch proto {
	case "grpc":
		server.Transport = grpctr.New(server.lis, opts...)
	case "http":
		server.Transport = httptr.New(server.lis, opts...)
	default:
		return nil, errInvalidListenTarget
	}
	return server, nil
}

func Dial(target string, opts ...option.DialOption) (ClientConn, error) {
	return DialContext(context.Background(), target, opts...)
}

func DialContext(ctx context.Context, target string,
	opts ...option.DialOption) (ClientConn, error) {

	s := strings.Split(target, "://")
	if len(s) != 2 {
		return nil, errInvalidDialTarget
	}
	proto, addr := s[0], s[1]
	switch proto {
	case "grpc":
		return grpctr.DialContext(ctx, addr, opts...)
	case "http":
		return httptr.DialContext(ctx, addr, opts...)
	default:
		return grpctr.DialContext(ctx, target, opts...)
	}
}
