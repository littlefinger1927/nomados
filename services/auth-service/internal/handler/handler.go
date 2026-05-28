package handler

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	authv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/auth/v1"
	commonv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/common/v1"
	sessionv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/session/v1"
	"github.com/nomados/nomados/services/auth-service/internal/service"
)

// AuthServiceHandler implements the AuthServiceServer gRPC interface.
type AuthServiceHandler struct {
	authv1.UnimplementedAuthServiceServer
	svc           *service.AuthService
	sessionClient sessionv1.SessionServiceClient
}

// NewAuthServiceHandler creates a new AuthServiceHandler.
func NewAuthServiceHandler(svc *service.AuthService, sessionClient sessionv1.SessionServiceClient) *AuthServiceHandler {
	return &AuthServiceHandler{
		svc:           svc,
		sessionClient: sessionClient,
	}
}

// Register handles the gRPC Register RPC.
func (h *AuthServiceHandler) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	result, err := h.svc.Register(ctx, req.Username, req.DevicePublicKey, req.DeviceAttestation)
	if err != nil {
		return nil, err
	}

	return &authv1.RegisterResponse{
		WebauthnChallenge: result.Challenge,
		UserId: &commonv1.UUID{
			Value: result.User.ID.String(),
		},
	}, nil
}

// RegisterVerify handles the gRPC RegisterVerify RPC.
func (h *AuthServiceHandler) RegisterVerify(ctx context.Context, req *authv1.RegisterVerifyRequest) (*authv1.RegisterVerifyResponse, error) {
	userID, err := uuid.Parse(req.UserId.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	if err := h.svc.VerifyRegistration(ctx, userID, req.CredentialResponse, req.DeviceSignature); err != nil {
		return nil, err
	}

	// Get the user's devices to find the device ID created during registration.
	devices, err := h.svc.GetDevicesForUser(ctx, userID)
	if err != nil || len(devices) == 0 {
		return nil, fmt.Errorf("failed to get device for user: %w", err)
	}
	deviceID := devices[0].ID.String()

	// Create a real session via the session-service.
	sessionResp, err := h.sessionClient.Create(ctx, &sessionv1.CreateSessionRequest{
		UserId:    &commonv1.UUID{Value: userID.String()},
		DeviceId:  &commonv1.UUID{Value: deviceID},
		IpHash:    "",
		RiskScore: 0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &authv1.RegisterVerifyResponse{
		AccessToken:  sessionResp.AccessToken,
		RefreshToken: sessionResp.RefreshToken,
		DeviceId: &commonv1.UUID{
			Value: deviceID,
		},
	}, nil
}

// Login handles the gRPC Login RPC.
func (h *AuthServiceHandler) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	user, challenge, err := h.svc.Login(ctx, req.DevicePublicKey)
	if err != nil {
		return nil, err
	}

	return &authv1.LoginResponse{
		WebauthnChallenge: challenge,
		UserId: &commonv1.UUID{
			Value: user.ID.String(),
		},
	}, nil
}

// LoginVerify handles the gRPC LoginVerify RPC.
func (h *AuthServiceHandler) LoginVerify(ctx context.Context, req *authv1.LoginVerifyRequest) (*authv1.LoginVerifyResponse, error) {
	userID, err := uuid.Parse(req.UserId.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Verify the assertion using the service layer (real WebAuthn verification)
	if err := h.svc.VerifyAssertion(ctx, userID, req.AssertionResponse, req.DeviceSignature); err != nil {
		return nil, err
	}

	// Get the user's devices to find the device ID.
	devices, err := h.svc.GetDevicesForUser(ctx, userID)
	if err != nil || len(devices) == 0 {
		return nil, fmt.Errorf("failed to get device for user: %w", err)
	}
	deviceID := devices[0].ID.String()

	// Create a real session via the session-service.
	sessionResp, err := h.sessionClient.Create(ctx, &sessionv1.CreateSessionRequest{
		UserId:    &commonv1.UUID{Value: userID.String()},
		DeviceId:  &commonv1.UUID{Value: deviceID},
		IpHash:    "",
		RiskScore: 0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &authv1.LoginVerifyResponse{
		AccessToken:  sessionResp.AccessToken,
		RefreshToken: sessionResp.RefreshToken,
		SessionId:    sessionResp.SessionId,
		DeviceId: &commonv1.UUID{
			Value: deviceID,
		},
	}, nil
}