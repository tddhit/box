package util

import (
	"net"
	"strings"

	"github.com/tddhit/tools/log"
)

func GetLocalAddr(listenAddr string) string {
	var host string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Panic(err)
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok &&
			ipnet.IP.To4() != nil &&
			!ipnet.IP.IsLoopback() {
			if isExternalIP(ipnet.IP) {
				continue
			} else {
				host = ipnet.IP.String()
				break
			}
		}
	}
	if host == "" {
		log.Panic("no suitable LocalAddr")
	}
	s := strings.Split(listenAddr, ":")
	if len(s) < 2 {
		log.Panicf("invalid listener:%s", listenAddr)
	}
	port := s[len(s)-1]
	return host + ":" + port
}

func isExternalIP(IP net.IP) bool {
	if IP.IsLoopback() || IP.IsLinkLocalMulticast() || IP.IsLinkLocalUnicast() {
		return false
	}
	if ip4 := IP.To4(); ip4 != nil {
		switch true {
		case ip4[0] == 10:
			return false
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			return false
		case ip4[0] == 192 && ip4[1] == 168:
			return false
		default:
			return true
		}
	}
	return false
}

func GetExternalAddr(listenAddr string) string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	s := strings.Split(conn.LocalAddr().String(), ":")
	if len(s) < 2 {
		log.Fatalf("invalid localAddr:%s\n", conn.LocalAddr().String())
	}
	host := s[0]
	s = strings.Split(listenAddr, ":")
	if len(s) < 2 {
		log.Fatalf("invalid listenAddr:%s\n", listenAddr)
	}
	port := s[len(s)-1]
	return host + ":" + port
}
