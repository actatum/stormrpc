# StormRPC ⚡

[![Go Report Card](https://goreportcard.com/badge/actatum/stormrpc)](https://goreportcard.com/report/actatum/stormrpc)
![Build Status](https://github.com/actatum/stormrpc/actions/workflows/main.yaml/badge.svg)
[![codecov](https://codecov.io/gh/actatum/stormrpc/branch/main/graph/badge.svg)](https://codecov.io/gh/actatum/stormrpc)
[![Godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/actatum/stormrpc)
[![Release](https://img.shields.io/github/release/actatum/stormrpc.svg)](https://github.com/actatum/stormrpc/releases/latest)

StormRPC is an abstraction or wrapper on [`NATS`] Request/Reply messaging capabilities.

It provides some convenient features including:

- **Middleware**

  Middleware are decorators around `HandlerFunc`s. Some middleware are available within the package including `RequestID`, `Tracing` (via OpenTelemetry) `Logger` and `Recoverer`.

- **Body encoding and decoding**

  Marshalling and unmarshalling request bodies to structs. JSON, Protobuf, and Msgpack are supported out of the box.

- **Deadline propagation**

  Request deadlines are propagated from client to server so both ends will stop processing once the deadline has passed.

- **Error propagation**

  Responses have an `Error` attribute and these are propagated across the wire without needing to tweak your request/response schemas.

## Installation

### Runtime Library

The runtime library package `github.com/actatum/stormrpc` contains common types like `stormrpc.Error`, `stormrpc.Client` and `stormrpc.Server`. If you aren't generating servers and clients from protobuf definitions you only need to import the stormrpc package.

```bash
go get github.com/actatum/stormrpc
```

### Code Generator

You need to install `go` and the `protoc` compiler on your system. Then, install the protoc plugins `protoc-gen-stormrpc` and `protoc-gen-go` to generate Go code.

```bash
go install github.com/actatum/stormrpc/cmd/protoc-gen-stormrpc@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

#### Using [Buf](https://buf.build/docs/introduction)

You'll need to initialize a buf mod file alongside your protobuf definitions and a buf gen file in the root of your project.

```
├── pb
│   ├── v1
│       ├── buf.yaml
│       ├── service.proto
├── main.go
├── buf.gen.yaml
├── go.mod
├── go.sum
└── .gitignore
```

`buf.yaml`

```yaml
version: v1
breaking:
  use:
    - FILE
lint:
  use:
    - DEFAULT
```

`buf.gen.yaml`

```yaml
version: v1
plugins:
  - plugin: go
    out: ./
    opt: paths=source_relative
  - plugin: stormrpc
    out: ./
    opt: paths=source_relative
```

To generate client and server stubs using buf, run the following command

```bash
buf generate
```

#### Using Protoc

To generate client and server stubs using protoc, run the following command

```bash
protoc --go_out=$output_dir --stormrpc_out=$output_dir $input_proto_file
```

Code generation examples can be found [here](https://github.com/actatum/stormrpc/tree/main/examples/protogen)

## Basic Usage

### Server

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/actatum/stormrpc"
	"github.com/nats-io/nats-server/v2/server"
)

func echo(ctx context.Context, req stormrpc.Request) stormrpc.Response {
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
	ns.Start()
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
	})
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
```

### Client

```go
package main

import (
  "context"
  "fmt"
  "log"
  "time"

  "github.com/actatum/stormrpc"
  "github.com/nats-io/nats.go"
)

func main() {
  client, err := stormrpc.NewClient(nats.DefaultURL)
  if err != nil {
    log.Fatal(err)
  }
  defer client.Close()

  ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
  defer cancel()

  r, err := stormrpc.NewRequest("echo", map[string]string{"hello": "me"})
  if err != nil {
    log.Fatal(err)
  }

  resp := client.Do(ctx, r)
  if resp.Err != nil {
    log.Fatal(resp.Err)
  }

  fmt.Println(resp.Header)

  var result map[string]string
  if err = resp.Decode(&result); err != nil {
    log.Fatal(err)
  }

  fmt.Printf("Result: %v\n", result)
}
```

[`nats.go`]: https://github.com/nats-io/nats.go
[`NATS`]: https://docs.nats.io/
