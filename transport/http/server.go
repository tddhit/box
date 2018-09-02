package http

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"

	"github.com/golang/protobuf/proto"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"

	"github.com/tddhit/box/interceptor"
	"github.com/tddhit/box/transport/common"
	"github.com/tddhit/box/transport/option"
	"github.com/tddhit/tools/log"
)

type HttpServer struct {
	*http.Server
	mux  *runtime.ServeMux
	lis  net.Listener
	opts option.ServerOptions
}

type ServiceDesc struct {
	*grpc.ServiceDesc
	Pattern map[string]runtime.Pattern
}

func New(lis net.Listener,
	opts ...option.ServerOption) *HttpServer {

	var ops option.ServerOptions
	for _, o := range opts {
		o(&ops)
	}
	s := &HttpServer{
		Server: &http.Server{},
		lis:    lis,
		opts:   ops,
	}
	if ops.GatewayMux != nil {
		s.mux = ops.GatewayMux
	} else {
		s.mux = runtime.NewServeMux()
	}
	s.Server.Handler = s.mux
	return s
}

func (s *HttpServer) Register(desc common.ServiceDesc,
	service interface{}) {

	if s.opts.GatewayMux != nil {
		return
	}
	sd := desc.Desc().(*ServiceDesc)
	ht := reflect.TypeOf(sd.HandlerType).Elem()
	st := reflect.TypeOf(service)
	if !st.Implements(ht) {
		log.Fatalf("Registerfound the handler of type %v that does not satisfy %v", st, ht)
	}
	s.register(sd, service)
}

func (s *HttpServer) register(sd *ServiceDesc, handler interface{}) {
	hv := reflect.ValueOf(handler)
	for _, method := range sd.ServiceDesc.Methods {
		m := method
		handlerFunc := func(w http.ResponseWriter, req *http.Request,
			pathParams map[string]string) {

			s.handlerFunc(w, req, pathParams, hv, m, sd.ServiceName)
		}
		s.mux.Handle("POST", sd.Pattern[method.MethodName], handlerFunc)
	}
}

func (s *HttpServer) handlerFunc(w http.ResponseWriter,
	req *http.Request, pathParams map[string]string,
	hv reflect.Value, method grpc.MethodDesc, serviceName string) {

	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()
	if cn, ok := w.(http.CloseNotifier); ok {
		go func(done <-chan struct{}, closed <-chan bool) {
			select {
			case <-done:
			case <-closed:
				cancel()
			}
		}(ctx.Done(), cn.CloseNotify())
	}
	inboundMarshaler, outboundMarshaler :=
		runtime.MarshalerForRequest(s.mux, req)
	rctx, err := runtime.AnnotateContext(ctx, s.mux, req)
	if err != nil {
		runtime.HTTPError(ctx, s.mux, outboundMarshaler, w, req, err)
		return
	}
	m := hv.MethodByName(method.MethodName)
	f := func(ctx context.Context, req interface{},
		info *common.UnaryServerInfo) (interface{}, error) {

		return handleReq(ctx, m, inboundMarshaler, req.(*http.Request), pathParams)
	}
	h := interceptor.ChainUnaryServerMiddleware(f, s.opts.UnaryMiddlewares...)
	info := &common.UnaryServerInfo{
		Server:     s,
		FullMethod: fmt.Sprintf("/%s/%s", serviceName, method.MethodName),
	}
	resp, err := h(rctx, req, info)
	runtime.ForwardResponseMessage(ctx, s.mux, outboundMarshaler, w,
		req, resp.(proto.Message), s.mux.GetForwardResponseOptions()...)
}

func handleReq(ctx context.Context, method reflect.Value,
	marshaler runtime.Marshaler, req *http.Request,
	pathParams map[string]string) (proto.Message, error) {

	reqType := method.Type().In(1).Elem()
	protoReq := reflect.New(reqType).Interface().(proto.Message)
	err := marshaler.NewDecoder(req.Body).Decode(protoReq)
	if err != nil && err != io.EOF {
		return nil, err
	}
	err = nil
	replies := method.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(protoReq),
	})
	reply := replies[0].Interface().(proto.Message)
	if replies[1].Interface() != nil {
		err = replies[1].Interface().(error)
	}
	return reply, err
}

func (s *HttpServer) Close() {
	if s.opts.FuncBeforeClose != nil {
		s.opts.FuncBeforeClose()
	}
	s.Server.Shutdown(context.Background())
	if s.opts.FuncAfterClose != nil {
		s.opts.FuncAfterClose()
	}
}
