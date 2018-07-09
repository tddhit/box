package interceptor

import (
	"context"

	"github.com/tddhit/box/stats"
	"github.com/tddhit/box/transport/common"
)

type UnaryHandler func(ctx context.Context, req interface{},
	info *common.UnaryServerInfo) (rsp interface{}, err error)

type Middleware func(UnaryHandler) UnaryHandler

func chain(h UnaryHandler, others []Middleware) UnaryHandler {
	for i := len(others) - 1; i >= 0; i-- {
		h = others[i](h)
	}
	return h
}

func Chain(h UnaryHandler, others ...Middleware) UnaryHandler {
	var ms = []Middleware{
		withStats,
	}
	ms = append(ms, others...)
	return chain(h, ms)
}

func withStats(next UnaryHandler) UnaryHandler {
	return func(ctx context.Context, req interface{},
		info *common.UnaryServerInfo) (interface{}, error) {

		stats.GlobalStats().Lock()
		stats.GlobalStats().Method[info.FullMethod]++
		stats.GlobalStats().Unlock()

		return next(ctx, req, info)
	}
}
