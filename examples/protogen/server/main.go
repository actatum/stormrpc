package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/actatum/stormrpc"
	"github.com/actatum/stormrpc/examples/protogen/pb"
	"github.com/nats-io/nats-server/v2/server"
)

type echoServer struct{}

func (s *echoServer) Echo(ctx context.Context, in *pb.EchoRequest) (*pb.EchoResponse, error) {
	h := stormrpc.HeadersFromContext(ctx)
	fmt.Printf("headers: %v\n", h)
	return &pb.EchoResponse{
		Message: in.GetMessage(),
	}, nil
}

func main() {
	ns, err := server.NewServer(&server.Options{
		Port: 40897,
	})
	if err != nil {
		log.Fatal(err)
	}
	go ns.Start()
	defer func() {
		ns.Shutdown()
		ns.WaitForShutdown()
	}()

	if !ns.ReadyForConnections(1 * time.Second) {
		log.Fatal("timeout waiting for nats server")
	}

	srv, err := stormrpc.NewServer(&stormrpc.ServerConfig{
		NatsURL: ns.ClientURL(),
		Name:    "echo",
	}, stormrpc.WithErrorHandler(logError))
	if err != nil {
		log.Fatal(err)
	}

	svc := &echoServer{}

	pb.RegisterEchoerServer(srv, svc)

	go func() {
		_ = srv.Run()
	}()
	log.Printf("👋 Listening on %v", srv.Subjects())

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
	log.Printf("💀 Shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = srv.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
}

func logError(ctx context.Context, err error) {
	log.Printf("Server Error: %v\n", err)
}
