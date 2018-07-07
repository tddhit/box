package naming

import (
	etcd "github.com/coreos/etcd/clientv3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/naming"
)

func NewResolver(ec *etcd.Client) *GRPCResolver {
	return &GRPCResolver{ec}
}

func RoundRobin(r naming.Resolver) grpc.Balancer {
	return grpc.RoundRobin(r)
}
