package common

import "google.golang.org/grpc"

// grpc.ServiceDesc is different from http.ServiceDesc, so define interface
// grpc shares UnaryServerInfo with http, so redefine
// only grpc support stream, so redefine

type ServiceDesc interface {
	Desc() interface{}
}

type StreamDesc grpc.StreamDesc

type UnaryServerInfo grpc.UnaryServerInfo

type StreamServerInfo grpc.StreamServerInfo

type ClientStream grpc.ClientStream

type ServerStream grpc.ServerStream
