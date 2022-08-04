package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/actatum/stormrpc"
	"github.com/actatum/stormrpc/examples/protogen/pb"
)

func main() {
	client, err := stormrpc.NewClient("nats://0.0.0.0:40897")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	c := pb.NewEchoerClient(client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	headers := map[string]string{"Authorization": "Bearer xy.eay"}
	out, err := c.Echo(ctx, &pb.EchoRequest{Message: "protogen"}, stormrpc.WithHeaders(headers))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Response: %s\n", out.GetMessage())
}
