package tracing

import (
	"io"

	opentracing "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
	config "github.com/uber/jaeger-client-go/config"
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
	tracer, closer, err := cfg.New(opt.service, config.Logger(jaeger.StdLogger))
	if err != nil {
		return nil, err
	}
	return &Tracer{
		Tracer: tracer,
		opt:    opt,
		closer: closer,
	}, nil
}
