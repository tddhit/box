package confcenter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	etcd "github.com/coreos/etcd/clientv3"
	yaml "gopkg.in/yaml.v2"

	"github.com/tddhit/tools/log"
)

type ConfCenter struct {
	opt options
	ec  *etcd.Client
	key string
}

func New(ec *etcd.Client, key string, opts ...Option) *ConfCenter {
	opt := defaultOption
	for _, o := range opts {
		o(&opt)
	}
	return &ConfCenter{
		opt: opt,
		ec:  ec,
		key: key,
	}
}

func (c *ConfCenter) MakeConf(conf interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.opt.timeout)
	defer cancel()

	var confBytes []byte
	if rsp, err := c.ec.Get(ctx, c.key); err != nil {
		log.Error(err)
		return err
	} else {
		if len(rsp.Kvs) == 0 || rsp.Kvs[0].Value == nil {
			log.Error("empty config.")
			return errors.New("empty config")
		}
		confBytes = rsp.Kvs[0].Value
		if err := yaml.Unmarshal(confBytes, conf); err != nil {
			log.Error(err)
			return err
		}
	}
	if c.opt.savePath != "" {
		file, err := os.OpenFile(c.opt.savePath,
			os.O_CREATE|os.O_TRUNC|os.O_WRONLY|os.O_SYNC, 0666)
		if err != nil {
			log.Error(err)
			return err
		}
		file.Write(confBytes)
		file.Sync()
		file.Close()
	}
	return nil
}

func (c *ConfCenter) Watch() (etcd.WatchChan, error) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	var watchC etcd.WatchChan
	go func() {
		watchC = c.ec.Watch(ctx, c.key, etcd.WithPrefix())
		done <- struct{}{}
	}()
	select {
	case <-time.After(c.opt.timeout):
		cancel()
		return nil, fmt.Errorf("watch %s timeout.\n", c.key)
	case <-done:
		log.Infof("watch %s success.\n", c.key)
	}
	return watchC, nil
}
