package main

import (
	"context"
	"fmt"
	"log"
	"stormrpc"
	"time"
)

func main() {
	client, err := stormrpc.NewClient("nats://0.0.0.0:40897")
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	r, err := stormrpc.NewRequest(ctx, "echo", map[string]string{"hello": "me"})
	if err != nil {
		log.Fatal(err)
	}

	resp := client.Do(r)
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
