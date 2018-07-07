package main

import (
	"context"
	"os"
	"time"

    pb "github.com/tddhit/box/example/pb"
	"github.com/tddhit/box/transport"
	"github.com/tddhit/tools/log"
)

func main() {
	{
		conn, err := transport.Dial(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()

		c := pb.NewExampleGrpcClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		reply, err := c.Echo(ctx, &pb.EchoRequest{Msg: "hello"})
		if err != nil {
			log.Fatalf("could not echo: %v", err)
		}
		log.Debug("Grpc Echo: ", reply.Msg)
	}
	{
		conn, err := transport.Dial(os.Args[2])
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()

		c := pb.NewExampleHttpClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 
				time.Second)
		defer cancel()
		reply, err := c.Echo(ctx, &pb.EchoRequest{Msg: "hello"})
		if err != nil {
			log.Fatalf("could not echo: %v", err)
		}
		log.Debug("Http Echo: ", reply.Msg)
	}
}
