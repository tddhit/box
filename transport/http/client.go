package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/tddhit/box/transport/option"
	"github.com/tddhit/tools/log"
)

type HttpClient struct {
	*http.Client
	addr string
	opt  option.DialOptions
}

func DialContext(ctx context.Context, target string,
	opts ...option.DialOption) (*HttpClient, error) {

	var opt option.DialOptions
	for _, o := range opts {
		o(&opt)
	}
	c := &HttpClient{
		Client: &http.Client{
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout:   500 * time.Millisecond,
					KeepAlive: time.Second,
				}).Dial,
				MaxIdleConns:    0,
				IdleConnTimeout: time.Second,
			},
			Timeout: 500 * time.Millisecond,
		},
		addr: target,
		opt:  opt,
	}
	return c, nil
}

func (c *HttpClient) Invoke(ctx context.Context, method string,
	args interface{}, reply interface{}, opts ...option.CallOption) error {

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(args); err != nil {
		log.Error(err)
		return err
	}
	url := fmt.Sprintf("http://%s%s", c.addr, method)
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		log.Error(err)
		return err
	}
	rsp, err := c.Client.Do(req)
	if err != nil {
		log.Error(err)
		return err
	}
	defer rsp.Body.Close()
	if err = json.NewDecoder(rsp.Body).Decode(reply); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (c *HttpClient) Close() {
}
