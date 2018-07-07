package main

import (
	"os"

	"github.com/tddhit/tools/log"
	"github.com/tddhit/wox"
	"github.com/tddhit/wox/option"
)

func main() {
	cfg := option.Client{
		HTTPVersion:     "2.0",
		ConnectTimeout:  1000,
		ReadTimeout:     1000,
		IdleConnTimeout: 1000,
		KeepAlive:       1000,
		MaxIdleConns:    10,
	}
	c := wox.NewClient(cfg, "127.0.0.1:18870")
	header := make(map[string][]string)
	header["Content-Type"] = append(header["Content-Type"], "application/json")
	rsp, err := c.Request("POST", "/ner/recognize", header, []byte(os.Args[1]))
	log.Debug(string(rsp), err)
	rsp, err = c.Request("POST", "/status", header, []byte(os.Args[1]))
	log.Debug(string(rsp), err)
}
