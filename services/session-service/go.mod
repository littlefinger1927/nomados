module github.com/nomados/nomados/services/session-service

go 1.25.0

require (
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.9.2
	github.com/nats-io/nats.go v1.38.0
	github.com/nomados/nomados/packages/auth-sdk v0.0.0
	github.com/nomados/nomados/packages/logging v0.0.0
	github.com/nomados/nomados/packages/shared-types v0.0.0
	google.golang.org/grpc v1.81.1
)

require (
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/nats-io/nkeys v0.4.9 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	github.com/nomados/nomados/packages/auth-sdk => ../../packages/auth-sdk
	github.com/nomados/nomados/packages/logging => ../../packages/logging
	github.com/nomados/nomados/packages/shared-types => ../../packages/shared-types
)
