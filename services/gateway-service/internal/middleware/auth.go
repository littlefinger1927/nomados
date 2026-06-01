package middleware

import (
	"context"
	"net/http"
	"strings"

	authsdk "github.com/nomados/nomados/packages/auth-sdk"
)

type contextKey string

const (
	UserIDKey    contextKey = "user_id"
	SessionIDKey contextKey = "session_id"
	DeviceIDKey  contextKey = "device_id"
)

// PublicPaths are routes that don't require authentication.
// grpc-gateway produces camelCase paths (registerVerify, loginVerify)
// while proto field names use snake_case (register_verify, login_verify).
// Both formats are included to match regardless of path style.
var PublicPaths = map[string]bool{
	"/v1/auth/register":        true,
	"/v1/auth/registerVerify":  true,
	"/v1/auth/register_verify": true,
	"/v1/auth/login":           true,
	"/v1/auth/loginVerify":     true,
	"/v1/auth/login_verify":    true,
}

func AuthMiddleware(validator *authsdk.TokenValidator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if PublicPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
			return
		}
		claims, err := validator.ValidateAccessToken(parts[1])
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, SessionIDKey, claims.SessionID)
		ctx = context.WithValue(ctx, DeviceIDKey, claims.DeviceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserID(ctx context.Context) string {
	if v, ok := ctx.Value(UserIDKey).(string); ok {
		return v
	}
	return ""
}

func GetSessionID(ctx context.Context) string {
	if v, ok := ctx.Value(SessionIDKey).(string); ok {
		return v
	}
	return ""
}

func GetDeviceID(ctx context.Context) string {
	if v, ok := ctx.Value(DeviceIDKey).(string); ok {
		return v
	}
	return ""
}