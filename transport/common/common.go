package common

import "google.golang.org/grpc"

type ServiceDesc interface {
	Desc() interface{}
}

type UnaryServerInfo grpc.UnaryServerInfo
