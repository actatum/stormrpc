go install ./cmd/protoc-gen-stormrpc

protoc --proto_path prototest -I=. prototest/*.proto \
    --stormrpc_out=./prototest/gen_out --go_out=paths=import:./prototest/gen_out