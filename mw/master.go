package mw

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/tddhit/box/confcenter"
	"github.com/tddhit/box/mw/common"
	"github.com/tddhit/tools/log"
)

type workerState int

const (
	workerAlive workerState = iota
	workerReload
	workerQuit
	workerCrash
)

func (s workerState) String() string {
	switch s {
	case workerAlive:
		return "alive"
	case workerReload:
		return "reload"
	case workerQuit:
		return "quit"
	case workerCrash:
		return "crash"
	default:
		return fmt.Sprintf("unknown worker state:%d", s)
	}
}

const (
	reasonStart  = "start"
	reasonReload = "reload"
	reasonCrash  = "crash"

	workerNum = 1
)

type master struct {
	addr      string
	pid       int
	pidPath   string
	uc        sync.Map // key: workerPID, value:UnixConn
	children  sync.Map // key: workerPID, value:state
	forkStats sync.Map
	forkWG    sync.WaitGroup
	forkC     chan string

	confCenter *confcenter.ConfCenter
}

func newMaster(ctx context.Context) *master {
	f := ctx.Value(mwKey{}).(*MW)
	return &master{
		addr:    f.masterAddr,
		pid:     os.Getpid(),
		pidPath: f.pidPath,
		forkC:   make(chan string),
	}
}

func (m *master) run() {
	m.savePID()
	if err := os.Setenv(common.FORK, "1"); err != nil {
		log.Fatal(err)
	}
	go m.handleFork()
	for i := 0; i < workerNum; i++ {
		m.forkC <- reasonStart
	}
	go m.watchConf()
	go m.watchWorker()
	go m.serve()

	log.Infof("MasterStart\tPid=%d", m.pid)
	signalC := make(chan os.Signal)
	signal.Notify(signalC)
	for {
		select {
		case sig := <-signalC:
			log.Infof("WatchSignal\tPid=%d\tSig=%s\n", m.pid, sig)
			switch sig {
			case syscall.SIGHUP:
				log.Reopen()
				m.reload()
			case syscall.SIGINT, syscall.SIGQUIT:
				m.graceful()
				fallthrough
			case syscall.SIGTERM:
				goto exit
			}
		}
	}
exit:
	m.close()
}

func (m *master) savePID() {
	f, err := os.OpenFile(m.pidPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0666)
	if err != nil {
		log.Fatal(err)
	}
	f.WriteString(strconv.Itoa(m.pid))
	f.Sync()
	f.Close()
}

func (m *master) removePID() {
	err := os.Remove(m.pidPath)
	if err != nil {
		log.Error(err)
	}
}

func (m *master) watchConf() {
	if m.confCenter == nil {
		return
	}
	watchC, err := m.confCenter.Watch()
	if err != nil {
		log.Error(err)
	}
	for range watchC {
		m.reload()
	}
}

func (m *master) watchWorker() {
	f := func(key, value interface{}) bool {
		pid := key.(int)
		state := value.(workerState)
		switch state {
		case workerCrash:
			m.forkC <- reasonCrash
			fallthrough
		case workerQuit:
			m.children.Delete(pid)
			m.uc.Delete(pid)
		}
		return true
	}
	tick := time.Tick(time.Second)
	for range tick {
		m.children.Range(f)
	}
}

func (m *master) modifyState(from, to workerState) {
	f := func(key, value interface{}) bool {
		pid := key.(int)
		state := value.(workerState)
		if state == from {
			m.children.Store(pid, to)
		}
		return true
	}
	m.children.Range(f)
}

func (m *master) handleFork() {
	for reason := range m.forkC {
		switch reason {
		case reasonStart:
		case reasonReload:
			m.modifyState(workerAlive, workerReload)
		case reasonCrash:
		default:
		}
		if _, err := m.fork(reason); err != nil {
			switch reason {
			case reasonStart:
				log.Fatal(reason, err)
			default:
				log.Error(reason, err)
			}
		}
		time.Sleep(time.Second)
	}
}

func (m *master) fork(reason string) (pid int, err error) {
	times, loaded := m.forkStats.LoadOrStore(reason, 1)
	if loaded {
		m.forkStats.Store(reason, times.(int)+1)
	}
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		log.Error(err)
		return
	}
	execSpec := &syscall.ProcAttr{
		Env: append(os.Environ(), "REASON="+reason, "PPID="+strconv.Itoa(m.pid)),
		Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd(),
			uintptr(fds[1])},
	}
	pid, err = syscall.ForkExec(os.Args[0], os.Args, execSpec)
	if err != nil {
		log.Error(err)
		return
	}
	file := os.NewFile(uintptr(fds[0]), "")
	conn, _ := net.FileConn(file)
	uc, _ := conn.(*net.UnixConn)
	m.uc.Store(pid, uc)
	m.children.Store(pid, workerAlive)
	syscall.Close(fds[1])

	m.forkWG.Add(1)
	go func() {
		m.waitWorker(pid)
		m.forkWG.Done()
	}()
	go m.readMsg(pid, uc)

	return
}

func (m *master) waitWorker(pid int) {
	p, _ := os.FindProcess(pid)
	state, _ := p.Wait()
	status := state.Sys().(syscall.WaitStatus)
	if status.ExitStatus() != 0 {
		m.children.Store(pid, workerCrash)
		log.Errorf("WorkerCrash\tPid=%d\tStatus=%d\n", pid, status.ExitStatus())
	} else {
		m.children.Store(pid, workerQuit)
		log.Infof("WorkerQuit\tPid=%d\n", pid)
	}
	/*
		log.Error(status.Exited())
		log.Error(status.ExitStatus())
		log.Error(status.Signaled())
		log.Error(status.Signal())
		log.Error(status.CoreDump())
		log.Error(status.Stopped())
		log.Error(status.Continued())
		log.Error(status.StopSignal())
	*/
}

func (m *master) readMsg(pid int, uc *net.UnixConn) {
	for {
		msg, err := readMsg(uc, "master", m.pid)
		if err != nil {
			break
		}
		switch msg.Typ {
		case msgTakeover:
			m.notifyWorker(&message{Typ: msgQuit}, workerReload)
			log.Infof("ReadMsg\tPid=%d\tMsg=%s\n", pid, msg.Typ)
		}
	}
}

func (m *master) reload() {
	for i := 0; i < workerNum; i++ {
		m.forkC <- reasonReload
	}
}

func (m *master) notifyWorker(msg *message, states ...workerState) {
	f := func(key, value interface{}) bool {
		pid := key.(int)
		curState := value.(workerState)
		match := false
		for _, state := range states {
			if curState == state {
				match = true
				break
			}
		}
		if !match {
			return true
		}
		if uc, ok := m.uc.Load(pid); !ok {
			log.Errorf("NotInUnixConn\tPid=%d\n", pid)
			return true
		} else {
			var buf bytes.Buffer
			gob.NewEncoder(&buf).Encode(msg)
			if _, _, err := uc.(*net.UnixConn).WriteMsgUnix(buf.Bytes(), nil, nil); err != nil {
				log.Warnf("WriteMsg\tPid=%d\tErr=%s\n", pid, err.Error())
			} else {
				log.Infof("WriteMsg\tPid=%d\tMsg=%s\n", pid, msg.Typ)
			}
		}
		return true
	}
	m.children.Range(f)
}

func (m *master) graceful() {
	m.notifyWorker(&message{Typ: msgQuit}, workerAlive, workerReload)
	m.forkWG.Wait()
}

func (m *master) close() {
	m.removePID()
	log.Infof("MasterEnd\tPid=%d\n", m.pid)
}

func (m *master) serve() {
	http.HandleFunc("/stats", m.doStats)
	if err := http.ListenAndServe(m.addr, nil); err != nil {
		log.Fatal(err)
	}
}

func (m *master) doStats(rsp http.ResponseWriter, req *http.Request) {
	var jsonRsp struct {
		Worker map[string]int `json:"worker"`
	}
	jsonRsp.Worker = make(map[string]int)
	m.forkStats.Range(func(key, value interface{}) bool {
		jsonRsp.Worker[key.(string)] = value.(int)
		return true
	})
	out, _ := json.Marshal(jsonRsp)
	rsp.Write(out)
}
