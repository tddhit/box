package interceptor

import (
	"github.com/tddhit/box/transport/common"
)

type StreamHandler func(srv interface{}, ss common.ServerStream,
	info *common.StreamServerInfo) error

type StreamServerMiddleware func(StreamHandler) StreamHandler

func ChainStreamServerMiddleware(h StreamHandler,
	others ...StreamServerMiddleware) StreamHandler {

	var ms = []StreamServerMiddleware{}
	ms = append(ms, others...)
	for i := len(ms) - 1; i >= 0; i-- {
		h = ms[i](h)
	}
	return h
}
