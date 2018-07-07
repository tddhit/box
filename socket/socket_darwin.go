package socket

import (
	"net"
	"os"
	"syscall"

	"github.com/tddhit/tools/log"
)

func Listen(addr string) (lis net.Listener, err error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		log.Error(err)
		return
	}
	sockaddr := &syscall.SockaddrInet4{Port: tcpAddr.Port}
	if ipv4 := tcpAddr.IP.To4(); ipv4 != nil {
		addr := [4]byte{ipv4[0], ipv4[1], ipv4[2], ipv4[3]}
		sockaddr.Addr = addr
	}
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_TCP)
	if err != nil {
		log.Error(err)
		return
	}
	err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	if err != nil {
		log.Error(err)
		return
	}
	err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEPORT, 1)
	if err != nil {
		log.Error(err)
		return
	}
	err = syscall.Bind(fd, sockaddr)
	if err != nil {
		log.Error(err)
		return
	}
	err = syscall.Listen(fd, 128)
	if err != nil {
		log.Error(err)
		return
	}
	file := os.NewFile(uintptr(fd), "")
	lis, err = net.FileListener(file)
	if err != nil {
		log.Error(err)
		return
	}
	return
}
