package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

const RequestIDKey = "request_id"

// RequestIDMiddleware generates a UUID v4 request ID for each incoming request.
// If the request already has an X-Request-ID header, it preserves it for
// distributed tracing. The request ID is set in the response header and
// stored in the request context for downstream use.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set the request ID in the response header
		w.Header().Set("X-Request-ID", requestID)

		// Store in context for downstream handlers
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID extracts the request ID from a context.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}