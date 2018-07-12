package mw

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/tddhit/box/confcenter"
	"github.com/tddhit/box/mw/common"
	"github.com/tddhit/box/transport"
	"github.com/tddhit/box/util"
	"github.com/tddhit/tools/log"
)

type mwKey struct{}

type MW struct {
	masterAddr string
	workerAddr string
	pidPath    string
	master     *master
	worker     *worker

	servers    []*transport.Server
	confCenter *confcenter.ConfCenter
}

func New(opts ...Option) *MW {
	f := &MW{}
	for _, opt := range opts {
		opt(f)
	}
	if f.masterAddr != "" {
		s := strings.Split(f.masterAddr, ":")
		if len(s) != 2 {
			log.Fatal("invalid masterAddr")
		}
		ip := s[0]
		if ip == "" {
			f.masterAddr = util.GetLocalAddr(f.masterAddr)
		}
	} else if f.servers != nil {
		f.masterAddr = getDefaultAddr(f.servers[len(f.servers)-1].Addr(), 2)
	}
	if f.workerAddr != "" {
		s := strings.Split(f.workerAddr, ":")
		if len(s) != 2 {
			log.Fatal("invalid workerAddr")
		}
		ip := s[0]
		if ip == "" {
			f.workerAddr = util.GetLocalAddr(f.workerAddr)
		}
	} else if f.servers != nil {
		f.workerAddr = getDefaultAddr(f.servers[len(f.servers)-1].Addr(), 1)
	}
	if f.pidPath == "" {
		name := strings.Split(os.Args[0], "/")
		if len(name) == 0 {
			log.Fatal("get pidPath fail:", os.Args[0])
		}
		f.pidPath = fmt.Sprintf("/var/%s.pid", name[len(name)-1])
	}
	baseCtx := context.Background()
	ctx := context.WithValue(baseCtx, mwKey{}, f)
	if os.Getenv(common.FORK) == "1" {
		f.worker = newWorker(ctx)
	} else {
		f.master = newMaster(ctx)
	}
	return f
}

func (f *MW) Go() {
	if os.Getenv(common.FORK) == "1" {
		f.worker.run()
	} else {
		f.master.run()
	}
}

func IsWorker() bool {
	return os.Getenv(common.FORK) == "1"
}

func Run(opts ...Option) {
	f := New(opts...)
	f.Go()
}

// get default masterAddr/workerAddr.
// eg. transportAddr:80, workerAddr:81, masterAddr:82
func getDefaultAddr(addr string, n int) string {
	a := strings.Split(addr, ":")
	port, _ := strconv.Atoi(a[len(a)-1])
	a[len(a)-1] = strconv.Itoa(port + n)
	return strings.Join(a, ":")
}
