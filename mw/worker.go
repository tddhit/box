package mw

import (
	"bytes"
	"context"
	"encoding/gob"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tddhit/box/socket"
	"github.com/tddhit/box/stats"
	"github.com/tddhit/box/transport"
	"github.com/tddhit/tools/log"
)

var (
	OK = []byte(`{"code":200}`)
)

type worker struct {
	addr string
	pid  int
	ppid int
	uc   *net.UnixConn
	wg   sync.WaitGroup

	servers []*transport.Server
}

func newWorker(ctx context.Context) *worker {
	f := ctx.Value(mwKey{}).(*MW)
	w := &worker{
		addr: f.workerAddr,
		pid:  os.Getpid(),

		servers: f.servers,
	}
	ppid := os.Getenv("PPID")
	w.ppid, _ = strconv.Atoi(ppid)
	file := os.NewFile(3, "")
	if conn, err := net.FileConn(file); err != nil {
		log.Fatal(err)
	} else {
		if uc, ok := conn.(*net.UnixConn); ok {
			w.uc = uc
		} else {
			log.Fatal(err)
		}
	}
	return w
}

func (w *worker) run() {
	go w.watchMaster()
	go w.watchSignal()
	go w.calcQPS()
	go w.serve()

	if w.servers != nil {
		for _, server := range w.servers {
			w.wg.Add(1)
			go func() {
				server.Serve()
				w.wg.Done()
			}()
			// make sure msgQuit can stop server when readMsg receive msg
			<-server.Started()
			server.RegisterAddr()
		}
	}

	time.Sleep(100 * time.Millisecond)
	go w.readMsg()

	reason := os.Getenv("REASON")
	if reason == reasonReload {
		if err := w.notifyMaster(&message{Typ: msgTakeover}); err == nil {
			log.Infof("WriteMsg\tPid=%d\tMsg=%s\n", w.pid, msgTakeover)
		}
	}
	log.Infof("WorkerStart\tPid=%d\tReason=%s\n", w.pid, reason)
	w.wg.Wait()
	log.Infof("WorkerEnd\tPid=%d\n", w.pid)
}

func (w *worker) calcQPS() {
	tick := time.Tick(time.Second)
	for range tick {
		stats.GlobalStats().Calculate()
	}
}

func (w *worker) watchMaster() {
	tick := time.Tick(1 * time.Second)
	for {
		select {
		case <-tick:
			if os.Getppid() != w.ppid {
				log.Fatal("MasterWorker is dead:", os.Getppid(), w.ppid)
			}
		}
	}
}

func (w *worker) watchSignal() {
	signalC := make(chan os.Signal)
	signal.Notify(signalC)
	for {
		select {
		case sig := <-signalC:
			log.Infof("WatchSignal\tPid=%d\tSig=%s\n", w.pid, sig.String())
		}
	}
}

func (w *worker) readMsg() {
	for {
		msg, err := readMsg(w.uc, "worker", w.pid)
		if err != nil {
			log.Fatal(err)
		}
		switch msg.Typ {
		case msgQuit:
			log.Infof("ReadMsg\tPid=%d\tMsg=%s\n", w.pid, msg.Typ)
			goto exit
		}
	}
exit:
	w.close()
}

func (w *worker) notifyMaster(msg *message) (err error) {
	var buf bytes.Buffer
	gob.NewEncoder(&buf).Encode(msg)
	if _, _, err = w.uc.WriteMsgUnix(buf.Bytes(), nil, nil); err != nil {
		log.Errorf("WriteMsg\tPid=%d\tErr=%s\n", w.pid, err.Error())
		return
	}
	return
}

func (w *worker) serve() {
	http.HandleFunc("/status", w.doStatus)
	http.HandleFunc("/stats", w.doStats)
	http.HandleFunc("/stats.html", w.doStatsHTML)
	//http.Handle("/metrics", promhttp.Handler())
	lis, err := socket.Listen(w.addr)
	if err != nil {
		log.Fatal(err)
	}
	srv := &http.Server{Handler: http.DefaultServeMux}
	if err := srv.Serve(lis); err != nil {
		log.Fatal(err)
	}
}

func (w *worker) doStatus(rsp http.ResponseWriter, req *http.Request) {
	rsp.Write(OK)
}

func (w *worker) doStats(rsp http.ResponseWriter, req *http.Request) {
	rsp.Header().Set("Content-Type", "application/json; charset=utf-8")
	rsp.Header().Set("Access-Control-Allow-Origin", "*")
	rsp.Write(stats.GlobalStats().Bytes())
}

func (w *worker) doStatsHTML(rsp http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		rsp.Write([]byte(err.Error()))
	}
	var (
		html string
		addr = req.FormValue("addr")
	)
	if addr != "" {
		html = strings.Replace(stats.GlobalStats().Html,
			"##ListenAddr##", addr, 1)
	} else {
		html = strings.Replace(stats.GlobalStats().Html,
			"##ListenAddr##", w.addr, 1)
	}
	rsp.Header().Set("Content-Type", "text/html; charset=utf-8")
	rsp.Write([]byte(html))
}

func (w *worker) close() {
	for _, s := range w.servers {
		go func(s *transport.Server) {
			s.UnegisterAddr()
			s.Close()
		}(s)
	}
}
