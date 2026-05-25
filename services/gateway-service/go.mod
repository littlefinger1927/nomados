module github.com/nomados/nomados/services/gateway-service

go 1.25.0

require (
	github.com/nomados/nomados/packages/auth-sdk v0.0.0
	golang.org/x/time v0.15.0
)

require github.com/golang-jwt/jwt/v5 v5.2.1 // indirect

replace github.com/nomados/nomados/packages/auth-sdk => ../../packages/auth-sdk
