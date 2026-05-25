module github.com/nomados/nomados/services/workspace-orchestrator

go 1.25.0

require (
	github.com/google/uuid v1.6.0
	github.com/nats-io/nats.go v1.38.0
	github.com/nomados/nomados/packages/logging v0.0.0
	github.com/nomados/nomados/packages/shared-types v0.0.0
	google.golang.org/grpc v1.81.1
)

require (
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/nats-io/nkeys v0.4.9 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	github.com/nomados/nomados/packages/logging => ../../packages/logging
	github.com/nomados/nomados/packages/shared-types => ../../packages/shared-types
)
