package mw

import (
	"github.com/tddhit/box/confcenter"
	"github.com/tddhit/box/transport"
)

type Option func(*MW)

func WithMasterAddr(a string) Option {
	return func(f *MW) {
		f.masterAddr = a
	}
}

func WithWorkerAddr(a string) Option {
	return func(f *MW) {
		f.workerAddr = a
	}
}

func WithPIDPath(p string) Option {
	return func(f *MW) {
		f.pidPath = p
	}
}

func WithConfCenter(c *confcenter.ConfCenter) Option {
	return func(f *MW) {
		f.confCenter = c
	}
}

func WithServer(s *transport.Server) Option {
	return func(f *MW) {
		f.servers = append(f.servers, s)
	}
}
