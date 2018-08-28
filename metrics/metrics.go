package metrics

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/tddhit/box/interceptor"
	"github.com/tddhit/box/transport/common"
)

var m *metrics

type metrics struct {
	count   *prometheus.CounterVec
	err     *prometheus.CounterVec
	latency *prometheus.HistogramVec
}

func init() {
	m = &metrics{
		count: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "request_count",
				Help: "the total number of requets",
			},
			[]string{"endpoint"},
		),
		err: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "request_error",
				Help: "the total number of error requets",
			},
			[]string{"endpoint"},
		),
		latency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "request_latency",
				Help:    "time per query",
				Buckets: []float64{10, 50, 100, 200, 500, 1000, 2000},
			},
			[]string{"endpoint"},
		),
	}
	prometheus.MustRegister(m.count)
	prometheus.MustRegister(m.err)
	prometheus.MustRegister(m.latency)
}

func Middleware(next interceptor.UnaryHandler) interceptor.UnaryHandler {
	return func(ctx context.Context, req interface{},
		info *common.UnaryServerInfo) (interface{}, error) {

		m.count.WithLabelValues(info.FullMethod).Inc()
		start := time.Now()
		rsp, err := next(ctx, req, info)
		elapse := float64(time.Since(start) / time.Millisecond)
		m.latency.WithLabelValues(info.FullMethod).Observe(elapse)
		if err != nil {
			m.err.WithLabelValues(info.FullMethod).Inc()
		}
		return rsp, err
	}
}
