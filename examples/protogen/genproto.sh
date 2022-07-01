go install ../../protoc-gen-stormrpc
protoc --go_out=./pb --stormrpc_out=./pb pb/echo.proto