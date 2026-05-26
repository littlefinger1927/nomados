package integration

import (
	"context"
	"testing"
	"time"

	authv1 "github.com/nomados/nomados/packages/shared-types/gen/auth/v1"
	commonv1 "github.com/nomados/nomados/packages/shared-types/gen/common/v1"
	sessionv1 "github.com/nomados/nomados/packages/shared-types/gen/session/v1"
)

// TestUserRegistration tests registering a new user via the auth service
// and verifying the response contains a valid user ID and challenge.
func TestUserRegistration(t *testing.T) {
	skipIfUnreachable(t, authServiceAddr, "auth")

	client, conn := newAuthClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, pubKey := generateTestKeyPair(t)

	resp, err := client.Register(ctx, &authv1.RegisterRequest{
		Username:          "integration-test-user-" + randomSuffix(),
		DevicePublicKey:   pubKey,
		DeviceAttestation: "test-attestation",
	})
	if err != nil {
		t.Fatalf("Register RPC failed: %v", err)
	}

	if resp.UserId == nil || resp.UserId.Value == "" {
		t.Error("expected non-empty user ID in response")
	}
	if len(resp.WebauthnChallenge) == 0 {
		t.Error("expected non-empty WebAuthn challenge in response")
	}
}

// TestLoginFlow tests the complete login flow:
// register a user, then verify the login challenge is returned.
func TestLoginFlow(t *testing.T) {
	skipIfUnreachable(t, authServiceAddr, "auth")

	client, conn := newAuthClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, pubKey := generateTestKeyPair(t)

	// Register a new user first
	regResp, err := client.Register(ctx, &authv1.RegisterRequest{
		Username:          "login-test-user-" + randomSuffix(),
		DevicePublicKey:   pubKey,
		DeviceAttestation: "test-attestation",
	})
	if err != nil {
		t.Fatalf("Register RPC failed: %v", err)
	}
	if regResp.UserId == nil || regResp.UserId.Value == "" {
		t.Fatal("expected non-empty user ID from registration")
	}

	// Now attempt login with the same device public key
	loginResp, err := client.Login(ctx, &authv1.LoginRequest{
		DevicePublicKey: pubKey,
	})
	if err != nil {
		t.Fatalf("Login RPC failed: %v", err)
	}

	if loginResp.UserId == nil || loginResp.UserId.Value == "" {
		t.Error("expected non-empty user ID in login response")
	}
	if len(loginResp.WebauthnChallenge) == 0 {
		t.Error("expected non-empty WebAuthn challenge in login response")
	}
}

// TestSessionRefresh tests the session lifecycle:
// create a session, refresh it, and verify the new token works.
func TestSessionRefresh(t *testing.T) {
	skipIfUnreachable(t, sessionServiceAddr, "session")

	client, conn := newSessionClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, pubKey := generateTestKeyPair(t)

	// Create a session (session proto uses plain strings for IDs)
	createResp, err := client.Create(ctx, &sessionv1.CreateSessionRequest{
		UserId:    "test-user-refresh-" + randomSuffix(),
		DeviceId:  "test-device-refresh-" + randomSuffix(),
		IpHash:    "test-ip-hash",
		RiskScore: 0,
	})
	if err != nil {
		t.Fatalf("CreateSession RPC failed: %v", err)
	}
	if createResp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if createResp.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}

	// Refresh the session using the refresh token
	refreshResp, err := client.Refresh(ctx, &sessionv1.RefreshSessionRequest{
		RefreshToken:    createResp.RefreshToken,
		DevicePublicKey: pubKey,
	})
	if err != nil {
		t.Fatalf("Refresh RPC failed: %v", err)
	}
	if refreshResp.AccessToken == "" {
		t.Error("expected non-empty access token after refresh")
	}
	if refreshResp.RefreshToken == "" {
		t.Error("expected non-empty refresh token after refresh")
	}

	// Verify the new access token works
	validateResp, err := client.Validate(ctx, &sessionv1.ValidateSessionRequest{
		AccessToken:     refreshResp.AccessToken,
		DevicePublicKey: pubKey,
	})
	if err != nil {
		t.Fatalf("Validate RPC failed after refresh: %v", err)
	}
	if !validateResp.Valid {
		t.Error("expected refreshed session to be valid")
	}
}

// TestSessionRevocation tests creating a session, revoking it,
// and verifying the revoked session fails validation.
func TestSessionRevocation(t *testing.T) {
	skipIfUnreachable(t, sessionServiceAddr, "session")

	client, conn := newSessionClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, pubKey := generateTestKeyPair(t)

	// Create a session
	createResp, err := client.Create(ctx, &sessionv1.CreateSessionRequest{
		UserId:    "test-user-revoke-" + randomSuffix(),
		DeviceId:  "test-device-revoke-" + randomSuffix(),
		IpHash:    "test-ip-hash",
		RiskScore: 0,
	})
	if err != nil {
		t.Fatalf("CreateSession RPC failed: %v", err)
	}

	// Validate the session works before revocation
	validateResp, err := client.Validate(ctx, &sessionv1.ValidateSessionRequest{
		AccessToken:     createResp.AccessToken,
		DevicePublicKey: pubKey,
	})
	if err != nil {
		t.Fatalf("Validate RPC failed before revocation: %v", err)
	}
	if !validateResp.Valid {
		t.Fatal("expected session to be valid before revocation")
	}

	// Revoke the session (session proto uses plain string for session ID)
	_, err = client.Revoke(ctx, &sessionv1.RevokeSessionRequest{
		SessionId: createResp.SessionId,
	})
	if err != nil {
		t.Fatalf("Revoke RPC failed: %v", err)
	}

	// Verify the revoked session is no longer valid
	validateResp, err = client.Validate(ctx, &sessionv1.ValidateSessionRequest{
		AccessToken:     createResp.AccessToken,
		DevicePublicKey: pubKey,
	})
	if err != nil {
		// An error from the service is acceptable - it means the session was properly revoked
		t.Logf("Validation of revoked session returned error (expected): %v", err)
		return
	}
	if validateResp.Valid {
		t.Error("expected revoked session to be invalid, but got valid=true")
	}
}

// TestDeviceBinding verifies that session tokens include device ID claims
// and that validation returns the correct device binding information.
func TestDeviceBinding(t *testing.T) {
	skipIfUnreachable(t, sessionServiceAddr, "session")

	client, conn := newSessionClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, pubKey := generateTestKeyPair(t)
	deviceID := "test-device-bind-" + randomSuffix()

	// Create a session bound to a specific device
	createResp, err := client.Create(ctx, &sessionv1.CreateSessionRequest{
		UserId:    "test-user-bind-" + randomSuffix(),
		DeviceId:  deviceID,
		IpHash:    "test-ip-hash",
		RiskScore: 0,
	})
	if err != nil {
		t.Fatalf("CreateSession RPC failed: %v", err)
	}

	// Validate with the correct device key
	validateResp, err := client.Validate(ctx, &sessionv1.ValidateSessionRequest{
		AccessToken:     createResp.AccessToken,
		DevicePublicKey: pubKey,
	})
	if err != nil {
		t.Fatalf("Validate RPC failed: %v", err)
	}
	if !validateResp.Valid {
		t.Error("expected session to be valid with correct device key")
	}
	if validateResp.DeviceId == "" {
		t.Error("expected device ID in validated session claims")
	}
	if validateResp.DeviceId != deviceID {
		t.Errorf("expected device ID %s, got %s", deviceID, validateResp.DeviceId)
	}
}

// unused import guard
var _ = commonv1.UUID{}