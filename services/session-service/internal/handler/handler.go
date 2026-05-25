package handler

import (
	"context"

	sessionv1 "github.com/nomados/nomados/packages/shared-types/gen/session/v1"
	commonv1 "github.com/nomados/nomados/packages/shared-types/gen/common/v1"
	"github.com/nomados/nomados/services/session-service/internal/service"
)

// SessionServiceHandler implements the SessionServiceServer gRPC interface.
type SessionServiceHandler struct {
	sessionv1.UnimplementedSessionServiceServer
	svc *service.SessionService
}

// NewSessionServiceHandler creates a new SessionServiceHandler.
func NewSessionServiceHandler(svc *service.SessionService) *SessionServiceHandler {
	return &SessionServiceHandler{
		svc: svc,
	}
}

// Create handles the gRPC Create RPC.
func (h *SessionServiceHandler) Create(ctx context.Context, req *sessionv1.CreateSessionRequest) (*sessionv1.CreateSessionResponse, error) {
	result, err := h.svc.CreateSession(ctx, req.UserId, req.DeviceId, req.IpHash, int(req.RiskScore))
	if err != nil {
		return nil, err
	}

	return &sessionv1.CreateSessionResponse{
		SessionId:    result.SessionID,
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt: &commonv1.Timestamp{
			Seconds: result.ExpiresAt.Unix(),
			Nanos:   int32(result.ExpiresAt.Nanosecond()),
		},
	}, nil
}

// Validate handles the gRPC Validate RPC.
func (h *SessionServiceHandler) Validate(ctx context.Context, req *sessionv1.ValidateSessionRequest) (*sessionv1.ValidateSessionResponse, error) {
	claims, err := h.svc.ValidateSession(ctx, req.AccessToken, req.DevicePublicKey)
	if err != nil {
		return &sessionv1.ValidateSessionResponse{
			Valid: false,
		}, nil
	}

	return &sessionv1.ValidateSessionResponse{
		Valid:     true,
		UserId:    claims.UserID,
		SessionId: claims.SessionID,
		DeviceId:  claims.DeviceID,
	}, nil
}

// Refresh handles the gRPC Refresh RPC.
func (h *SessionServiceHandler) Refresh(ctx context.Context, req *sessionv1.RefreshSessionRequest) (*sessionv1.RefreshSessionResponse, error) {
	result, err := h.svc.RefreshSession(ctx, req.RefreshToken, req.DevicePublicKey)
	if err != nil {
		return nil, err
	}

	return &sessionv1.RefreshSessionResponse{
		SessionId:    result.SessionID,
		AccessToken:   result.AccessToken,
		RefreshToken:  result.RefreshToken,
		ExpiresAt: &commonv1.Timestamp{
			Seconds: result.ExpiresAt.Unix(),
			Nanos:   int32(result.ExpiresAt.Nanosecond()),
		},
	}, nil
}

// Revoke handles the gRPC Revoke RPC.
func (h *SessionServiceHandler) Revoke(ctx context.Context, req *sessionv1.RevokeSessionRequest) (*commonv1.Empty, error) {
	if err := h.svc.RevokeSession(ctx, req.SessionId); err != nil {
		return nil, err
	}
	return &commonv1.Empty{}, nil
}