go install ./protoc-gen-stormrpc
protoc --proto_path prototest -I=. prototest/test.proto \
    --stormrpc_out=./prototest/gen_out --go_out=./prototest