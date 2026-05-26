module github.com/nomados/nomados/tests/integration

go 1.25.0

require (
	github.com/google/uuid v1.6.0
	github.com/nomados/nomados/packages/auth-sdk v0.0.0
	github.com/nomados/nomados/packages/shared-types v0.0.0
	golang.org/x/crypto v0.48.0
	google.golang.org/grpc v1.81.1
)

require (
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	github.com/nomados/nomados/packages/auth-sdk => ../../packages/auth-sdk
	github.com/nomados/nomados/packages/logging => ../../packages/logging
	github.com/nomados/nomados/packages/shared-types => ../../packages/shared-types
	github.com/nomados/nomados/services/auth-service => ../../services/auth-service
	github.com/nomados/nomados/services/session-service => ../../services/session-service
	github.com/nomados/nomados/services/vault-service => ../../services/vault-service
	github.com/nomados/nomados/services/workspace-orchestrator => ../../services/workspace-orchestrator
)
