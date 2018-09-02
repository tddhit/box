package interceptor

import (
	"context"

	"github.com/tddhit/box/stats"
	"github.com/tddhit/box/transport/common"
)

type UnaryHandler func(ctx context.Context, req interface{},
	info *common.UnaryServerInfo) (rsp interface{}, err error)

type UnaryServerMiddleware func(UnaryHandler) UnaryHandler

func ChainUnaryServerMiddleware(h UnaryHandler,
	others ...UnaryServerMiddleware) UnaryHandler {

	var ms = []UnaryServerMiddleware{
		withStats,
	}
	ms = append(ms, others...)
	for i := len(ms) - 1; i >= 0; i-- {
		h = ms[i](h)
	}
	return h
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
