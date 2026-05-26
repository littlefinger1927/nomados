module github.com/nomados/nomados/services/gateway-service

go 1.25.0

require (
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0
	github.com/nomados/nomados/packages/auth-sdk v0.0.0
	github.com/nomados/nomados/packages/shared-types v0.0.0
	golang.org/x/time v0.15.0
	google.golang.org/grpc v1.81.1
)

require (
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260414002931-afd174a4e478 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260414002931-afd174a4e478 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	github.com/nomados/nomados/packages/auth-sdk => ../../packages/auth-sdk
	github.com/nomados/nomados/packages/shared-types => ../../packages/shared-types
)
