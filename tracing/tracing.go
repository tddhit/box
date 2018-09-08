package tracing

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"strings"
	"unsafe"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	jaeger "github.com/uber/jaeger-client-go"
	config "github.com/uber/jaeger-client-go/config"
	"google.golang.org/grpc/metadata"

	"github.com/tddhit/box/interceptor"
	"github.com/tddhit/box/transport/common"
	"github.com/tddhit/tools/log"
)

var t *Tracer

type Tracer struct {
	opentracing.Tracer
	opt       options
	closer    io.Closer
	marshaler *jsonpb.Marshaler
}

func Init(opts ...Option) error {
	if t != nil {
		log.Panic("tracer has been initialized")
	}
	opt := defaultOption
	for _, o := range opts {
		o(&opt)
	}
	cfg := &config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: opt.agentAddr,
		},
	}
	tracer, closer, err := cfg.New(opt.service)
	if err != nil {
		return nil
	}
	opentracing.SetGlobalTracer(tracer)
	t = &Tracer{
		Tracer:    tracer,
		opt:       opt,
		closer:    closer,
		marshaler: &jsonpb.Marshaler{EnumsAsInts: true},
	}
	return nil
}

func ServerMiddleware(next interceptor.UnaryHandler) interceptor.UnaryHandler {
	return func(ctx context.Context, req interface{},
		info *common.UnaryServerInfo) (interface{}, error) {

		if t == nil {
			Init()
		}
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}
		spanCtx, err := t.Extract(opentracing.TextMap, mdReaderWriter{&md})
		if err != nil && err != opentracing.ErrSpanContextNotFound {
			log.Error(err)
		}
		span := t.StartSpan(
			info.FullMethod,
			ext.RPCServerOption(spanCtx),
			ext.SpanKindRPCServer,
		)
		ctx = opentracing.ContextWithSpan(ctx, span)
		rsp, err := next(ctx, req, info)
		buf := &bytes.Buffer{}
		if err := t.marshaler.Marshal(buf, req.(proto.Message)); err == nil {
			span.LogKV("req", buf.String())
		}
		if rsp != nil {
			face := (*iface)(unsafe.Pointer(&rsp))
			if face.data != 0 {
				buf.Reset()
				if err := t.marshaler.Marshal(buf, rsp.(proto.Message)); err == nil {
					span.LogKV("rsp", buf.String())
				}
			}
		}
		if err != nil {
			span.LogKV("err", err.Error())
		}
		span.Finish()
		return rsp, err
	}
}

func ClientMiddleware(next interceptor.UnaryInvoker) interceptor.UnaryInvoker {
	return func(ctx context.Context, method string,
		req, reply interface{}) error {

		if t == nil {
			Init()
		}
		var parentCtx opentracing.SpanContext
		if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
			parentCtx = parentSpan.Context()
		}
		clientSpan := t.StartSpan(
			method,
			opentracing.ChildOf(parentCtx),
			ext.SpanKindRPCClient,
		)
		defer clientSpan.Finish()

		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		} else {
			md = md.Copy()
		}

		err := t.Inject(clientSpan.Context(),
			opentracing.TextMap, mdReaderWriter{&md})
		if err != nil {
			log.Error(err)
		}
		ctx = metadata.NewOutgoingContext(ctx, md)
		return next(ctx, method, req, reply)
	}
}

func Release() {
	if t == nil {
		log.Panic("uninitiated tracer ")
	}
	t.closer.Close()
}

func TraceIDFromContext(ctx context.Context) string {
	if t == nil {
		Init()
	}
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		if spanCtx := span.Context(); spanCtx != nil {
			if c, ok := spanCtx.(jaeger.SpanContext); ok {
				return c.TraceID().String()
			}
		}
	}
	return ""
}

func Execute(ctx context.Context, operationName string, f func(ctx context.Context)) {
	if t == nil {
		Init()
	}
	span, spanCtx := opentracing.StartSpanFromContext(ctx, operationName)
	f(spanCtx)
	span.Finish()
}

func Start(ctx context.Context,
	operationName string) (opentracing.Span, context.Context) {

	if t == nil {
		Init()
	}
	return opentracing.StartSpanFromContext(ctx, operationName)
}

func Stop(span opentracing.Span) {
	if t == nil {
		Init()
	}
	span.Finish()
}

type mdReaderWriter struct {
	*metadata.MD
}

func (w mdReaderWriter) Set(key, val string) {
	key = strings.ToLower(key)
	if strings.HasSuffix(key, "-bin") {
		val = string(base64.StdEncoding.EncodeToString([]byte(val)))
	}
	(*w.MD)[key] = append((*w.MD)[key], val)
}

func (w mdReaderWriter) ForeachKey(handler func(key, val string) error) error {
	for k, vals := range *w.MD {
		for _, v := range vals {
			if err := handler(k, v); err != nil {
				return err
			}
		}
	}
	return nil
}

type iface struct {
	typ  uintptr
	data uintptr
}
