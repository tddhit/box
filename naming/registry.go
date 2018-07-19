package naming

import (
	"context"
	"os"
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

func (r *Registry) Register(serviceName, addr string) context.CancelFunc {
	done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	key := serviceName + "/" + addr
	go func() {
		rsp, err := r.ec.Grant(ctx, r.opt.ttl)
		if err != nil {
			log.Fatal(err)
			return
		}
		_, err = r.ec.Get(ctx, key)
		if err != nil && err != rpctypes.ErrKeyNotFound {
			log.Fatal(err)
			return
		}
		ch, err := r.ec.KeepAlive(ctx, rsp.ID)
		if err != nil {
			log.Fatal(err)
			return
		}
		if ch == nil {
			log.Fatal("ch is nil")
			return
		}
		if _, err = r.ec.Put(ctx, key, addr, etcd.WithLease(rsp.ID)); err != nil {
			log.Fatal(err)
			return
		}
		log.Infof("registry success. Pid=%d key=%s leaseID=%d",
			os.Getpid(), key, rsp.ID)
		go func() {
			for range ch {
			}
			log.Warnf("registry close. Pid=%d key=%s leaseID=%d",
				os.Getpid(), key, rsp.ID)
		}()
		done <- struct{}{}
	}()
	select {
	case <-time.After(r.opt.timeout):
		cancel()
		log.Fatalf("registry timeout. Pid=%d\tkey=%s\n", os.Getpid(), key)
	case <-done:
		//log.Infof("registry success. Pid=%d\tkey=%s\n", os.Getpid(), key)
	}
	return cancel
}

func (r *Registry) TTL() int64 {
	return r.opt.ttl
}
