package naming

import (
	"context"
	"time"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/etcdserver/api/v3rpc/rpctypes"

	"github.com/tddhit/tools/log"
)

type Registry struct {
	opt registryOptions
	ec  *etcd.Client
}

func NewRegistry(ec *etcd.Client, opts ...RegistryOption) *Registry {

	opt := defaultRegistryOption
	for _, o := range opts {
		o(&opt)
	}
	return &Registry{
		opt: opt,
		ec:  ec,
	}
}

func (r *Registry) Register(key, addr string) context.CancelFunc {
	done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		key := key + "/" + addr
		rsp, err := r.ec.Grant(ctx, r.opt.ttl)
		if err != nil {
			log.Error(err)
			return
		}
		_, err = r.ec.Get(ctx, key)
		if err != nil && err != rpctypes.ErrKeyNotFound {
			log.Error(err)
			return
		}
		ch, err := r.ec.KeepAlive(ctx, rsp.ID)
		if err != nil {
			log.Error(err)
			return
		}
		if ch == nil {
			log.Error("ch is nil")
			return
		}
		if _, err = r.ec.Put(ctx, key+"/"+addr, addr,
			etcd.WithLease(rsp.ID)); err != nil {
			log.Error(err)
			return
		}
		go func() {
			for range ch {
			}
			log.Warn("registry keepalive close.")
		}()
		done <- struct{}{}
	}()
	select {
	case <-time.After(r.opt.timeout):
		cancel()
		log.Fatalf("registry %s/%s timeout.\n", key, addr)
	case <-done:
		log.Infof("registry success:%s/%s\n", key, addr)
	}
	return cancel
}

func (r *Registry) TTL() int64 {
	return r.opt.ttl
}
