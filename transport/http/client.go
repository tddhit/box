package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"

	"github.com/tddhit/box/transport/common"
	"github.com/tddhit/box/transport/option"
	"github.com/tddhit/tools/log"
)

type HttpClient struct {
	*http.Client
	addr        string
	opt         option.DialOptions
	marshaler   *jsonpb.Marshaler
	unmarshaler *jsonpb.Unmarshaler
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
		addr:        target,
		opt:         opt,
		marshaler:   &jsonpb.Marshaler{EnumsAsInts: true},
		unmarshaler: &jsonpb.Unmarshaler{AllowUnknownFields: true},
	}
	return c, nil
}

func (c *HttpClient) Invoke(ctx context.Context, method string,
	args interface{}, reply interface{}, opts ...option.CallOption) error {

	var buf bytes.Buffer
	if err := c.marshaler.Marshal(&buf, args.(proto.Message)); err != nil {
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
	if err = c.unmarshaler.Unmarshal(rsp.Body, reply.(proto.Message)); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (c *HttpClient) Close() {
}

func (c *HttpClient) NewStream(ctx context.Context, desc common.ServiceDesc, i int,
	method string, opts ...option.CallOption) (common.ClientStream, error) {

	return nil, errors.New("http does not support stream.")
}
