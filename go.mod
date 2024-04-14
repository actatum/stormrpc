module github.com/actatum/stormrpc

go 1.21

// replace github.com/nats-io/nats.go => ../nats.go

replace github.com/nats-io/nats.go => github.com/actatum/nats.go v1.31.1-0.20231207185944-7538a5cd8e3f

require (
	github.com/google/uuid v1.3.0
	github.com/nats-io/nats-server/v2 v2.10.7
	github.com/nats-io/nats.go v1.31.1-0.20231201130123-4af26aae2522
	github.com/vmihailenco/msgpack/v5 v5.3.5
	go.opentelemetry.io/otel v1.14.0
	go.opentelemetry.io/otel/sdk v1.7.0
	go.opentelemetry.io/otel/trace v1.14.0
	google.golang.org/protobuf v1.33.0
)

require (
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/minio/highwayhash v1.0.2 // indirect
	github.com/nats-io/jwt/v2 v2.5.3 // indirect
	github.com/nats-io/nkeys v0.4.6 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	golang.org/x/crypto v0.17.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
	golang.org/x/time v0.5.0 // indirect
)

// v1.31.1-0.20231207012943-0e824f8b9d26
