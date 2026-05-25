package service

import (
	"context"
	"testing"
	"time"

	authsdk "github.com/nomados/nomados/packages/auth-sdk"
	"github.com/nomados/nomados/services/session-service/internal/repository"
)

func TestNewSessionService(t *testing.T) {
	svc := NewSessionService(nil, nil, "test-secret")
	if svc == nil {
		t.Fatal("expected non-nil SessionService")
	}
}

func TestCreateSessionTokenGeneration(t *testing.T) {
	// Test that the token validator can generate tokens for session creation
	validator := authsdk.NewTokenValidator("test-secret")

	userID := "550e8400-e29b-41d4-a716-446655440000"
	sessionID := "660e8400-e29b-41d4-a716-446655440001"
	deviceID := "770e8400-e29b-41d4-a716-446655440002"

	accessToken, err := validator.GenerateAccessToken(userID, sessionID, deviceID, 15*time.Minute)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}
	if accessToken == "" {
		t.Error("expected non-empty access token")
	}

	refreshToken, err := validator.GenerateAccessToken(userID, sessionID, deviceID, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}
	if refreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
}

func TestValidateSessionTokenClaims(t *testing.T) {
	validator := authsdk.NewTokenValidator("test-secret")

	userID := "550e8400-e29b-41d4-a716-446655440000"
	sessionID := "660e8400-e29b-41d4-a716-446655440001"
	deviceID := "770e8400-e29b-41d4-a716-446655440002"

	token, err := validator.GenerateAccessToken(userID, sessionID, deviceID, 15*time.Minute)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := validator.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected user_id %s, got %s", userID, claims.UserID)
	}
	if claims.SessionID != sessionID {
		t.Errorf("expected session_id %s, got %s", sessionID, claims.SessionID)
	}
	if claims.DeviceID != deviceID {
		t.Errorf("expected device_id %s, got %s", deviceID, claims.DeviceID)
	}
}

func TestValidateSessionExpiredToken(t *testing.T) {
	validator := authsdk.NewTokenValidator("test-secret")

	userID := "550e8400-e29b-41d4-a716-446655440000"
	sessionID := "660e8400-e29b-41d4-a716-446655440001"
	deviceID := "770e8400-e29b-41d4-a716-446655440002"

	// Generate a token that's already expired
	token, err := validator.GenerateAccessToken(userID, sessionID, deviceID, -1*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = validator.ValidateAccessToken(token)
	if err == nil {
		t.Error("expected error for expired token, got nil")
	}
}

func TestValidateSessionWrongSecret(t *testing.T) {
	validator1 := authsdk.NewTokenValidator("secret-1")
	validator2 := authsdk.NewTokenValidator("secret-2")

	userID := "550e8400-e29b-41d4-a716-446655440000"
	sessionID := "660e8400-e29b-41d4-a716-446655440001"
	deviceID := "770e8400-e29b-41d4-a716-446655440002"

	token, _ := validator1.GenerateAccessToken(userID, sessionID, deviceID, 15*time.Minute)

	_, err := validator2.ValidateAccessToken(token)
	if err == nil {
		t.Error("expected error for token signed with wrong secret, got nil")
	}
}

func TestSessionClaimsStructure(t *testing.T) {
	claims := &SessionClaims{
		UserID:    "user-123",
		SessionID: "session-456",
		DeviceID:  "device-789",
	}

	if claims.UserID != "user-123" {
		t.Errorf("expected UserID user-123, got %s", claims.UserID)
	}
	if claims.SessionID != "session-456" {
		t.Errorf("expected SessionID session-456, got %s", claims.SessionID)
	}
	if claims.DeviceID != "device-789" {
		t.Errorf("expected DeviceID device-789, got %s", claims.DeviceID)
	}
}

func TestSessionResponseStructure(t *testing.T) {
	now := time.Now()
	resp := &SessionResponse{
		SessionID:    "session-123",
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresAt:     now,
	}

	if resp.SessionID != "session-123" {
		t.Errorf("expected SessionID session-123, got %s", resp.SessionID)
	}
	if resp.AccessToken != "access-token" {
		t.Errorf("expected AccessToken access-token, got %s", resp.AccessToken)
	}
	if resp.RefreshToken != "refresh-token" {
		t.Errorf("expected RefreshToken refresh-token, got %s", resp.RefreshToken)
	}
	if resp.ExpiresAt != now {
		t.Error("expected ExpiresAt to match")
	}
}

func TestValidateSessionMissingDeviceID(t *testing.T) {
	validator := authsdk.NewTokenValidator("test-secret")

	// Generate a token without device ID
	token, err := validator.GenerateAccessToken("user-1", "session-1", "", 15*time.Minute)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := validator.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	// ValidateSession would reject this because DeviceID is empty
	if claims.DeviceID != "" {
		t.Errorf("expected empty DeviceID, got %s", claims.DeviceID)
	}
}

func TestCreateSessionWithNilRepo(t *testing.T) {
	// Service with nil repo should return error on DB operations
	// We test this by using a nil PostgresRepository — the pool nil check
	// inside repository methods handles the graceful error path.
	repo := repository.NewPostgresRepository(nil)
	svc := NewSessionService(repo, nil, "test-secret")
	_, err := svc.CreateSession(context.Background(), "user-1", "device-1", "hash-1", 0)
	if err == nil {
		t.Error("expected error when repo pool is nil, got nil")
	}
}

func TestRevokeSessionWithNilRepo(t *testing.T) {
	repo := repository.NewPostgresRepository(nil)
	svc := NewSessionService(repo, nil, "test-secret")
	err := svc.RevokeSession(context.Background(), "session-1")
	if err == nil {
		t.Error("expected error when repo pool is nil, got nil")
	}
}

func TestRevokeAllUserSessionsWithNilRepo(t *testing.T) {
	repo := repository.NewPostgresRepository(nil)
	svc := NewSessionService(repo, nil, "test-secret")
	err := svc.RevokeAllUserSessions(context.Background(), "user-1")
	if err == nil {
		t.Error("expected error when repo pool is nil, got nil")
	}
}