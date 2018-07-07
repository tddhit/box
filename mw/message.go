package mw

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"

	"github.com/tddhit/tools/log"
)

type msgType int

const (
	msgTakeover msgType = iota // worker->master
	msgQuit                    // master->worker
)

func (m msgType) String() string {
	switch m {
	case msgTakeover:
		return "takeover"
	case msgQuit:
		return "quit"
	default:
		return fmt.Sprintf("unknown msg type:%d", m)
	}
}

type message struct {
	Typ   msgType
	Value interface{}
}

func readMsg(conn *net.UnixConn, id string, pid int) (*message, error) {
	buf := make([]byte, 1024)
	msg := &message{}
	if _, _, _, _, err := conn.ReadMsgUnix(buf, nil); err != nil {
		log.Warnf("ReadMsg\tId=%s\tPid=%d\tErr=%s\n", id, pid, err.Error())
		return nil, err
	}
	if err := gob.NewDecoder(bytes.NewBuffer(buf)).Decode(msg); err != nil {
		return nil, err
	}
	return msg, nil
}
