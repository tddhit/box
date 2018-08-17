package interceptor

import (
	"context"
)

type UnaryInvoker func(ctx context.Context,
	method string, req, reply interface{}) error

type UnaryClientMiddleware func(UnaryInvoker) UnaryInvoker

func chainUnaryClient(h UnaryInvoker, others []UnaryClientMiddleware) UnaryInvoker {
	for i := len(others) - 1; i >= 0; i-- {
		h = others[i](h)
	}
	return h
}

func ChainUnaryClient(h UnaryInvoker, others ...UnaryClientMiddleware) UnaryInvoker {
	var ms = []UnaryClientMiddleware{}
	ms = append(ms, others...)
	return chainUnaryClient(h, ms)
}
