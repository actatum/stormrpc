// Package main provides the executable function for the protoc-gen-stormrpc binary.
package main

import (
	stormrpcgen "github.com/actatum/stormrpc/internal/gen"
	"google.golang.org/protobuf/compiler/protogen"
)

func main() {
	protogen.Options{}.Run(func(gen *protogen.Plugin) error {
		for _, f := range gen.Files {
			if !f.Generate {
				continue
			}
			stormrpcgen.GenerateFile(gen, f)
		}
		return nil
	})
}
