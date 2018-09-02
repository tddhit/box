package interceptor

import (
	"context"
)

type UnaryInvoker func(ctx context.Context,
	method string, req, reply interface{}) error

type UnaryClientMiddleware func(UnaryInvoker) UnaryInvoker

func ChainUnaryClientMiddleware(h UnaryInvoker,
	others ...UnaryClientMiddleware) UnaryInvoker {

	var ms = []UnaryClientMiddleware{}
	ms = append(ms, others...)
	for i := len(ms) - 1; i >= 0; i-- {
		h = ms[i](h)
	}
	return h
}
