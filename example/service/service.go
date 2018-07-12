package service

import (
	"context"
	pb "github.com/tddhit/box/example/pb"
)

type service struct{}

func NewService() *service {
	return &service{}
}

func (h *service) Echo(ctx context.Context, in *pb.EchoRequest) (*pb.EchoReply, error) {
	return &pb.EchoReply{Msg: in.Msg}, nil
}
