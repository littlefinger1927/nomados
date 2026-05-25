package handler

import (
	"context"

	"github.com/google/uuid"
	authv1 "github.com/nomados/nomados/packages/shared-types/gen/auth/v1"
	commonv1 "github.com/nomados/nomados/packages/shared-types/gen/common/v1"
	"github.com/nomados/nomados/services/auth-service/internal/service"
)

// AuthServiceHandler implements the AuthServiceServer gRPC interface.
type AuthServiceHandler struct {
	authv1.UnimplementedAuthServiceServer
	svc *service.AuthService
}

// NewAuthServiceHandler creates a new AuthServiceHandler.
func NewAuthServiceHandler(svc *service.AuthService) *AuthServiceHandler {
	return &AuthServiceHandler{
		svc: svc,
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
		return nil, err
	}

	if err := h.svc.VerifyRegistration(ctx, userID, req.CredentialResponse, req.DeviceSignature); err != nil {
		return nil, err
	}

	// TODO: Generate real access/refresh tokens using auth-sdk
	// For now, return placeholder tokens
	return &authv1.RegisterVerifyResponse{
		AccessToken:  "placeholder_access_token",
		RefreshToken: "placeholder_refresh_token",
		DeviceId: &commonv1.UUID{
			Value: userID.String(), // placeholder
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
	// TODO: Implement real WebAuthn assertion verification and token generation
	// For now, return placeholder tokens
	userID, err := uuid.Parse(req.UserId.Value)
	if err != nil {
		return nil, err
	}

	// Verify the assertion (stub — always succeeds)
	if err := verifyAssertionCredential(req.AssertionResponse, nil); err != nil {
		return nil, err
	}

	return &authv1.LoginVerifyResponse{
		AccessToken:  "placeholder_access_token",
		RefreshToken: "placeholder_refresh_token",
		SessionId: &commonv1.UUID{
			Value: uuid.New().String(), // placeholder
		},
		DeviceId: &commonv1.UUID{
			Value: userID.String(), // placeholder
		},
	}, nil
}