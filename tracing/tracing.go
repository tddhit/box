package tracing

import (
	"context"
	"encoding/base64"
	"io"
	"strings"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	config "github.com/uber/jaeger-client-go/config"
	"google.golang.org/grpc/metadata"

	"github.com/tddhit/box/interceptor"
	"github.com/tddhit/box/transport/common"
	"github.com/tddhit/tools/log"
)

type Tracer struct {
	opentracing.Tracer
	opt    options
	closer io.Closer
}

func New(opts ...Option) (*Tracer, error) {
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
		return nil, err
	}
	opentracing.SetGlobalTracer(tracer)
	return &Tracer{
		Tracer: tracer,
		opt:    opt,
		closer: closer,
	}, nil
}

func ServerMiddleware(tracer opentracing.Tracer) interceptor.UnaryServerMiddleware {
	return func(next interceptor.UnaryHandler) interceptor.UnaryHandler {
		return func(ctx context.Context, req interface{},
			info *common.UnaryServerInfo) (interface{}, error) {

			md, ok := metadata.FromIncomingContext(ctx)
			if !ok {
				md = metadata.New(nil)
			}
			spanCtx, err := tracer.Extract(opentracing.TextMap, mdReaderWriter{&md})
			if err != nil && err != opentracing.ErrSpanContextNotFound {
				log.Error(err)
			}
			span := tracer.StartSpan(
				info.FullMethod,
				ext.RPCServerOption(spanCtx),
				ext.SpanKindRPCServer,
			)
			defer span.Finish()

			ctx = opentracing.ContextWithSpan(ctx, span)
			return next(ctx, req, info)
		}
	}
}

func ClientMiddleware(tracer opentracing.Tracer) interceptor.UnaryClientMiddleware {
	return func(next interceptor.UnaryInvoker) interceptor.UnaryInvoker {
		return func(ctx context.Context, method string,
			req, reply interface{}) error {

			var parentCtx opentracing.SpanContext
			if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
				parentCtx = parentSpan.Context()
			}
			clientSpan := tracer.StartSpan(
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

			err := tracer.Inject(clientSpan.Context(), opentracing.TextMap, mdReaderWriter{&md})
			if err != nil {
				log.Error(err)
			}
			ctx = metadata.NewOutgoingContext(ctx, md)
			return next(ctx, method, req, reply)
		}
	}
}

func Execute(ctx context.Context, operationName string, f func()) context.Context {
	span, spanCtx := opentracing.StartSpanFromContext(ctx, operationName)
	f()
	span.Finish()
	return spanCtx
}

func Start(ctx context.Context, operationName string) opentracing.Span {
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		return opentracing.GlobalTracer().StartSpan(operationName,
			opentracing.ChildOf(span.Context()))
	}
	return opentracing.GlobalTracer().StartSpan(operationName)
}

func Stop(ctx context.Context, span opentracing.Span) context.Context {
	span.Finish()
	return opentracing.ContextWithSpan(ctx, span)
}

func (t *Tracer) Close() {
	t.closer.Close()
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
