package service

import (
	"context"
	"fmt"

	sessionv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/session/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// SessionValidator validates session tokens by calling the session service.
type SessionValidator struct {
	client sessionv1.SessionServiceClient
	conn   *grpc.ClientConn
}

// NewSessionValidator creates a new SessionValidator by connecting to the session service
// at the given address.
func NewSessionValidator(sessionAddr string) (*SessionValidator, error) {
	conn, err := grpc.NewClient(sessionAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to session service: %w", err)
	}
	return &SessionValidator{
		client: sessionv1.NewSessionServiceClient(conn),
		conn:   conn,
	}, nil
}

// Validate validates a session token and returns the user ID of the authenticated user.
// It calls the session service's Validate RPC.
func (v *SessionValidator) Validate(ctx context.Context, accessToken string) (userID string, err error) {
	resp, err := v.client.Validate(ctx, &sessionv1.ValidateSessionRequest{
		AccessToken: accessToken,
	})
	if err != nil {
		return "", fmt.Errorf("session validation failed: %w", err)
	}

	if !resp.GetValid() {
		return "", fmt.Errorf("session token is not valid")
	}

	uid := resp.GetUserId()
	if uid == nil || uid.GetValue() == "" {
		return "", fmt.Errorf("session validation response missing user ID")
	}

	return uid.GetValue(), nil
}

// Close closes the gRPC connection to the session service.
func (v *SessionValidator) Close() error {
	if v.conn != nil {
		return v.conn.Close()
	}
	return nil
}