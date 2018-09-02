package interceptor

import (
	"context"

	"github.com/tddhit/box/transport/common"
)

type StreamInvoker func(ctx context.Context, desc *common.StreamDesc,
	method string) (common.ClientStream, error)

type StreamClientMiddleware func(StreamInvoker) StreamInvoker

func ChainStreamClientMiddleware(h StreamInvoker,
	others ...StreamClientMiddleware) StreamInvoker {

	var ms = []StreamClientMiddleware{}
	ms = append(ms, others...)
	for i := len(ms) - 1; i >= 0; i-- {
		h = ms[i](h)
	}
	return h
}
