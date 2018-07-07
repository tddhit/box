package tracing

import "os"

var defaultOption = options{
	service:   os.Args[0],
	agentAddr: "127.0.0.1:6831",
}

type options struct {
	service   string
	agentAddr string
}

type Option func(*options)

func WithService(service string) Option {
	return func(o *options) {
		o.service = service
	}
}

func WithAgentAddr(addr string) Option {
	return func(o *options) {
		o.agentAddr = addr
	}
}
