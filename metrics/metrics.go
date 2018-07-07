package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	opt options
	QPS *prometheus.GaugeVec
	EPS *prometheus.GaugeVec   // error per second
	TPQ *prometheus.SummaryVec // time per query
}

func New(opts ...Option) *Metrics {
	opt := defaultOption
	for _, o := range opts {
		o(&opt)
	}
	m := &Metrics{
		opt: opt,
	}
	if opt.useQPS {
		m.QPS = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "QPS",
				Help: "query per second",
			},
			[]string{"endpoint"},
		)
		prometheus.MustRegister(m.QPS)
	}
	if opt.useEPS {
		m.EPS = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "EPS",
				Help: "error per second",
			},
			[]string{"endpoint"},
		)
		prometheus.MustRegister(m.EPS)
	}
	if opt.useTPQ {
		m.TPQ = prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: "TPQ",
				Help: "time per query",
			},
			[]string{"endpoint"},
		)
		prometheus.MustRegister(m.TPQ)
	}
	return m
}
