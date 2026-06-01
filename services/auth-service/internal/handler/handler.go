package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	authv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/auth/v1"
	commonv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/common/v1"
	sessionv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/session/v1"
	"github.com/nomados/nomados/services/auth-service/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

// AddCredential handles the gRPC AddCredential RPC.
func (h *AuthServiceHandler) AddCredential(ctx context.Context, req *authv1.AddCredentialRequest) (*authv1.AddCredentialResponse, error) {
	userID, err := protoUUIDToUUID(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	device, challenge, err := h.svc.AddCredential(ctx, userID, req.GetDeviceName())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add credential: %v", err)
	}

	return &authv1.AddCredentialResponse{
		WebauthnChallenge: challenge,
		DeviceId:          uuidToProtoUUID(device.ID),
	}, nil
}

// ListCredentials handles the gRPC ListCredentials RPC.
func (h *AuthServiceHandler) ListCredentials(ctx context.Context, req *authv1.ListCredentialsRequest) (*authv1.ListCredentialsResponse, error) {
	userID, err := protoUUIDToUUID(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	devices, err := h.svc.ListCredentials(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list credentials: %v", err)
	}

	creds := make([]*authv1.CredentialInfo, 0, len(devices))
	for _, d := range devices {
		creds = append(creds, &authv1.CredentialInfo{
			DeviceId:        uuidToProtoUUID(d.ID),
			DeviceName:      d.Name,
			AttestationType: d.Attestation,
			CreatedAt:       d.CreatedAt.Unix(),
			LastSeen:        lastSeenToUnix(d.LastSeen),
		})
	}

	return &authv1.ListCredentialsResponse{Credentials: creds}, nil
}

// RemoveCredential handles the gRPC RemoveCredential RPC.
func (h *AuthServiceHandler) RemoveCredential(ctx context.Context, req *authv1.RemoveCredentialRequest) (*commonv1.Empty, error) {
	userID, err := protoUUIDToUUID(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}
	deviceID, err := protoUUIDToUUID(req.GetDeviceId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid device_id: %v", err)
	}

	if err := h.svc.RemoveCredential(ctx, userID, deviceID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove credential: %v", err)
	}

	return &commonv1.Empty{}, nil
}

// protoUUIDToUUID converts a proto UUID to a Go uuid.UUID.
func protoUUIDToUUID(pb *commonv1.UUID) (uuid.UUID, error) {
	if pb == nil {
		return uuid.Nil, fmt.Errorf("uuid is nil")
	}
	return uuid.Parse(pb.GetValue())
}

// uuidToProtoUUID converts a Go uuid.UUID to a proto UUID.
func uuidToProtoUUID(id uuid.UUID) *commonv1.UUID {
	return &commonv1.UUID{Value: id.String()}
}

// lastSeenToUnix converts a *time.Time to a Unix timestamp, returning 0 for nil.
func lastSeenToUnix(t *time.Time) int64 {
	if t == nil {
		return 0
	}
	return t.Unix()
}