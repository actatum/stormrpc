package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"stormrpc"
	"syscall"
	"time"

	"github.com/nats-io/nats-server/v2/server"
)

func echo(req stormrpc.Request) stormrpc.Response {
	var b any
	if err := req.Decode(&b); err != nil {
		return stormrpc.NewErrorResponse(req.Reply, err)
	}

	resp, err := stormrpc.NewResponse(req.Reply, b)
	if err != nil {
		return stormrpc.NewErrorResponse(req.Reply, err)
	}

	return resp
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

	srv, err := stormrpc.NewServer("echo", ns.ClientURL())
	if err != nil {
		log.Fatal(err)
	}
	srv.Handle("echo", echo)

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
