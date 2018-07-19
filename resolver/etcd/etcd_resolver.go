/*
 *
 * Copyright 2017 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Package etcd implements a etcd resolver to be installed as the default resolver
// in grpc.
package etcd

import (
	"errors"
	"strings"
	"sync"
	"time"

	etcd "github.com/coreos/etcd/clientv3"
	"golang.org/x/net/context"
	"google.golang.org/grpc/resolver"

	"github.com/tddhit/tools/log"
)

func init() {
	resolver.Register(NewBuilder())
}

const (
	defaultFreq = time.Minute * 30
)

// NewBuilder creates a etcdBuilder which is used to factory etcd resolvers.
func NewBuilder() resolver.Builder {
	return &etcdBuilder{freq: defaultFreq}
}

type etcdBuilder struct {
	// frequency of polling the etcd server.
	freq time.Duration
}

// Build creates and starts a etcd resolver that watches the name resolution of the target.
func (b *etcdBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOption) (resolver.Resolver, error) {
	if target.Authority == "" {
		return nil, errors.New("no etcd endpoints")
	}
	endpoints := strings.Split(target.Authority, ",")
	cfg := etcd.Config{
		Endpoints:   endpoints,
		DialTimeout: 2 * time.Second,
	}
	ec, err := etcd.New(cfg)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	d := &etcdResolver{
		ec:          ec,
		freq:        b.freq,
		serviceName: target.Endpoint,
		cc:          cc,
		t:           time.NewTimer(0),
		rn:          make(chan struct{}, 1),
	}

	if err := d.watchEtcd(); err != nil {
		return nil, err
	}
	d.wg.Add(1)
	go d.watcher()
	return d, nil
}

// Scheme returns the naming scheme of this resolver builder, which is "etcd".
func (b *etcdBuilder) Scheme() string {
	return "etcd"
}

// etcdResolver watches for the name resolution update for a non-IP target.
type etcdResolver struct {
	ec          *etcd.Client
	freq        time.Duration
	serviceName string
	ctx         context.Context
	cancel      context.CancelFunc
	cc          resolver.ClientConn
	// rn channel is used by ResolveNow() to force an immediate resolution of the target.
	rn chan struct{}
	t  *time.Timer
	// wg is used to enforce Close() to return after the watcher() goroutine has finished.
	// Otherwise, data race will be possible. [Race Example] in etcd_resolver_test we
	// replace the real lookup functions with mocked ones to facilitate testing.
	// If Close() doesn't wait for watcher() goroutine finishes, race detector sometimes
	// will warns lookup (READ the lookup function pointers) inside watcher() goroutine
	// has data race with replaceNetFunc (WRITE the lookup function pointers).
	wg sync.WaitGroup
}

// ResolveNow invoke an immediate resolution of the target that this etcdResolver watches.
func (d *etcdResolver) ResolveNow(opt resolver.ResolveNowOption) {
	select {
	case d.rn <- struct{}{}:
	default:
	}
}

// Close closes the etcdResolver.
func (d *etcdResolver) Close() {
	d.cancel()
	d.wg.Wait()
	d.t.Stop()
}

func (d *etcdResolver) watchEtcd() error {
	var (
		watchC etcd.WatchChan
		done   = make(chan struct{})
	)
	d.ctx, d.cancel = context.WithCancel(context.Background())
	go func() {
		watchC = d.ec.Watch(d.ctx, d.serviceName, etcd.WithPrefix())
		select { //avoid goroutine leak
		case done <- struct{}{}:
		default:
		}
	}()
	select {
	case <-time.After(time.Second):
		log.Errorf("resolver watch timeout. target=%s", d.serviceName)
		d.cancel()
		return d.ctx.Err()
	case <-done:
		log.Infof("resolver watch success. target=%s", d.serviceName)
	}
	go func() {
		for rsp := range watchC {
			log.Infof("WatchEvent\t%s\n", d.serviceName)
			for _, event := range rsp.Events {
				log.Infof("WatchEvent\tType=%d\tKey=%s\tValue=%s\n",
					event.Type, string(event.Kv.Key), string(event.Kv.Value))
				d.ResolveNow(resolver.ResolveNowOption{})
			}
		}
	}()
	return nil
}

func (d *etcdResolver) watcher() {
	defer d.wg.Done()
	for {
		select {
		case <-d.ctx.Done():
			return
		case <-d.t.C:
		case <-d.rn:
		}
		result, err := d.lookup()
		// Next lookup should happen after an interval defined by d.freq.
		d.t.Reset(d.freq)
		if err != nil {
			log.Error(err)
		} else {
			log.Debug(result)
			d.cc.NewAddress(result)
		}
	}
}

func (d *etcdResolver) lookup() ([]resolver.Address, error) {
	var addrs []resolver.Address
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	if rsp, err := d.ec.Get(ctx, d.serviceName, etcd.WithPrefix()); err != nil {
		return nil, err
	} else {
		for _, kv := range rsp.Kvs {
			addrs = append(addrs, resolver.Address{Addr: string(kv.Value)})
		}
	}
	return addrs, nil
}
