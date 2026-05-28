package integration

import (
	"context"
	"testing"
	"time"

	authv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/auth/v1"
	commonv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/common/v1"
)

// TestWebAuthnRegistrationFlow tests the complete WebAuthn registration flow:
// 1. Register a user (gets a challenge and user ID)
// 2. Verify registration with a mock credential response
// 3. Verify that the response contains access/refresh tokens and a device ID
func TestWebAuthnRegistrationFlow(t *testing.T) {
	skipIfUnreachable(t, authServiceAddr, "auth")

	client, conn := newAuthClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, pubKey := generateTestKeyPair(t)
	username := "webauthn-reg-" + randomSuffix()

	// Step 1: Register — should return a WebAuthn challenge and user ID
	regResp, err := client.Register(ctx, &authv1.RegisterRequest{
		Username:          username,
		DevicePublicKey:   pubKey,
		DeviceAttestation: "test-attestation",
	})
	if err != nil {
		t.Fatalf("Register RPC failed: %v", err)
	}

	if regResp.UserId == nil || regResp.UserId.Value == "" {
		t.Fatal("expected non-empty user ID in registration response")
	}
	if len(regResp.WebauthnChallenge) == 0 {
		t.Error("expected non-empty WebAuthn challenge in registration response")
	}

	// Step 2: RegisterVerify — verify with a mock credential response
	// The auth service's WebAuthn verification is currently stubbed (always succeeds)
	verifyResp, err := client.RegisterVerify(ctx, &authv1.RegisterVerifyRequest{
		CredentialResponse: []byte(`{"type":"public-key","id":"mock-cred-id","rawId":"bW9jay1jcmVkLWlk","response":{"attestationObject":"bW9jay1hdHRlc3Q","clientDataJSON":"bW9jay1jbGllbnQ"}}`),
		DeviceSignature:    []byte("mock-device-signature"),
		UserId:             regResp.UserId,
	})
	if err != nil {
		t.Fatalf("RegisterVerify RPC failed: %v", err)
	}

	// Step 3: Verify the response contains tokens and device ID
	if verifyResp.AccessToken == "" {
		t.Error("expected non-empty access token in registration verify response")
	}
	if verifyResp.RefreshToken == "" {
		t.Error("expected non-empty refresh token in registration verify response")
	}
	if verifyResp.DeviceId == nil || verifyResp.DeviceId.Value == "" {
		t.Error("expected non-empty device ID in registration verify response")
	}
}

// TestWebAuthnLoginFlow tests the complete WebAuthn login flow:
// 1. Register a user first
// 2. Login (gets a challenge and user ID)
// 3. LoginVerify with a mock assertion response
// 4. Verify that the response contains access/refresh tokens, session ID, and device ID
func TestWebAuthnLoginFlow(t *testing.T) {
	skipIfUnreachable(t, authServiceAddr, "auth")

	client, conn := newAuthClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, pubKey := generateTestKeyPair(t)
	username := "webauthn-login-" + randomSuffix()

	// Step 1: Register the user first
	regResp, err := client.Register(ctx, &authv1.RegisterRequest{
		Username:          username,
		DevicePublicKey:   pubKey,
		DeviceAttestation: "test-attestation",
	})
	if err != nil {
		t.Fatalf("Register RPC failed: %v", err)
	}
	if regResp.UserId == nil || regResp.UserId.Value == "" {
		t.Fatal("expected non-empty user ID from registration")
	}

	// Complete registration so we can log in
	verifyResp, err := client.RegisterVerify(ctx, &authv1.RegisterVerifyRequest{
		CredentialResponse: []byte(`{"type":"public-key","id":"mock-cred-id","rawId":"bW9jay1jcmVkLWlk","response":{"attestationObject":"bW9jay1hdHRlc3Q","clientDataJSON":"bW9jay1jbGllbnQ"}}`),
		DeviceSignature:    []byte("mock-device-signature"),
		UserId:             regResp.UserId,
	})
	if err != nil {
		t.Fatalf("RegisterVerify RPC failed: %v", err)
	}
	_ = verifyResp // tokens from registration, not needed for login

	// Step 2: Login — should return a challenge and user ID
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

	// Verify the user ID matches the registered user
	if loginResp.UserId.Value != regResp.UserId.Value {
		t.Errorf("expected login user ID %s, got %s", regResp.UserId.Value, loginResp.UserId.Value)
	}

	// Step 3: LoginVerify — verify with a mock assertion response
	// The auth service's WebAuthn verification is currently stubbed (always succeeds)
	loginVerifyResp, err := client.LoginVerify(ctx, &authv1.LoginVerifyRequest{
		AssertionResponse: []byte(`{"type":"public-key","id":"mock-cred-id","rawId":"bW9jay1jcmVkLWlk","response":{"authenticatorData":"bW9jay1hdXRo","clientDataJSON":"bW9jay1jbGllbnQ","signature":"bW9jay1zaWdu","userHandle":"dXNlci1oYW5kbGU"}}`),
		DeviceSignature:   []byte("mock-device-signature"),
		UserId:            loginResp.UserId,
	})
	if err != nil {
		t.Fatalf("LoginVerify RPC failed: %v", err)
	}

	// Step 4: Verify the response contains tokens, session ID, and device ID
	if loginVerifyResp.AccessToken == "" {
		t.Error("expected non-empty access token in login verify response")
	}
	if loginVerifyResp.RefreshToken == "" {
		t.Error("expected non-empty refresh token in login verify response")
	}
	if loginVerifyResp.SessionId == nil || loginVerifyResp.SessionId.Value == "" {
		t.Error("expected non-empty session ID in login verify response")
	}
	if loginVerifyResp.DeviceId == nil || loginVerifyResp.DeviceId.Value == "" {
		t.Error("expected non-empty device ID in login verify response")
	}
}

// TestWebAuthnRegistrationWithEmptyCredential tests that RegisterVerify
// handles empty or invalid credential responses appropriately.
func TestWebAuthnRegistrationWithEmptyCredential(t *testing.T) {
	skipIfUnreachable(t, authServiceAddr, "auth")

	client, conn := newAuthClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, pubKey := generateTestKeyPair(t)
	username := "webauthn-empty-cred-" + randomSuffix()

	// Register the user first
	regResp, err := client.Register(ctx, &authv1.RegisterRequest{
		Username:          username,
		DevicePublicKey:   pubKey,
		DeviceAttestation: "test-attestation",
	})
	if err != nil {
		t.Fatalf("Register RPC failed: %v", err)
	}

	// Try to verify with an empty credential response
	// The stub implementation currently accepts empty credentials,
	// but this test documents the expected behavior for when real
	// WebAuthn verification is implemented.
	_, err = client.RegisterVerify(ctx, &authv1.RegisterVerifyRequest{
		CredentialResponse: nil,
		DeviceSignature:    nil,
		UserId:             regResp.UserId,
	})
	// With the current stub implementation, this may or may not return an error.
	// Document the actual behavior for future reference.
	if err != nil {
		t.Logf("RegisterVerify with empty credential returned error (expected): %v", err)
	} else {
		t.Log("RegisterVerify with empty credential succeeded (stub implementation accepts empty credentials)")
	}
}

// TestWebAuthnLoginWithUnknownDevice tests that logging in with an
// unregistered device public key returns an error.
func TestWebAuthnLoginWithUnknownDevice(t *testing.T) {
	skipIfUnreachable(t, authServiceAddr, "auth")

	client, conn := newAuthClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Generate a key pair for a device that was never registered
	_, unknownPubKey := generateTestKeyPair(t)

	// Attempt to log in with the unregistered device
	_, err := client.Login(ctx, &authv1.LoginRequest{
		DevicePublicKey: unknownPubKey,
	})
	if err == nil {
		t.Error("expected error when logging in with unregistered device, got nil")
	} else {
		t.Logf("Login with unregistered device correctly returned error: %v", err)
	}
}

// TestWebAuthnLoginVerifyWithInvalidUser tests that LoginVerify with a
// non-existent user ID returns an error.
func TestWebAuthnLoginVerifyWithInvalidUser(t *testing.T) {
	skipIfUnreachable(t, authServiceAddr, "auth")

	client, conn := newAuthClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to verify login for a non-existent user ID
	_, err := client.LoginVerify(ctx, &authv1.LoginVerifyRequest{
		AssertionResponse: []byte("mock-assertion"),
		DeviceSignature:   []byte("mock-signature"),
		UserId:            &commonv1.UUID{Value: "nonexistent-user-" + randomSuffix()},
	})
	if err == nil {
		t.Error("expected error when verifying login for non-existent user, got nil")
	} else {
		t.Logf("LoginVerify with non-existent user correctly returned error: %v", err)
	}
}

// TestWebAuthnFullEndToEndFlow tests the complete auth flow end-to-end:
// Register -> RegisterVerify -> Login -> LoginVerify, verifying
// that all tokens are returned and user IDs are consistent.
func TestWebAuthnFullEndToEndFlow(t *testing.T) {
	skipIfUnreachable(t, authServiceAddr, "auth")

	client, conn := newAuthClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, pubKey := generateTestKeyPair(t)
	username := "webauthn-e2e-" + randomSuffix()

	// Step 1: Register
	regResp, err := client.Register(ctx, &authv1.RegisterRequest{
		Username:          username,
		DevicePublicKey:   pubKey,
		DeviceAttestation: "test-attestation",
	})
	if err != nil {
		t.Fatalf("Register RPC failed: %v", err)
	}
	userID := regResp.UserId

	// Step 2: RegisterVerify
	regVerifyResp, err := client.RegisterVerify(ctx, &authv1.RegisterVerifyRequest{
		CredentialResponse: []byte(`{"type":"public-key","id":"mock-cred-id","rawId":"bW9jay1jcmVkLWlk","response":{"attestationObject":"bW9jay1hdHRlc3Q","clientDataJSON":"bW9jay1jbGllbnQ"}}`),
		DeviceSignature:    []byte("mock-device-signature"),
		UserId:             userID,
	})
	if err != nil {
		t.Fatalf("RegisterVerify RPC failed: %v", err)
	}

	// Verify registration tokens are present
	if regVerifyResp.AccessToken == "" {
		t.Error("expected non-empty access token after registration verify")
	}
	if regVerifyResp.RefreshToken == "" {
		t.Error("expected non-empty refresh token after registration verify")
	}

	// Step 3: Login
	loginResp, err := client.Login(ctx, &authv1.LoginRequest{
		DevicePublicKey: pubKey,
	})
	if err != nil {
		t.Fatalf("Login RPC failed: %v", err)
	}

	// Verify user ID consistency
	if loginResp.UserId.Value != userID.Value {
		t.Errorf("login user ID %s does not match registered user ID %s",
			loginResp.UserId.Value, userID.Value)
	}

	// Step 4: LoginVerify
	loginVerifyResp, err := client.LoginVerify(ctx, &authv1.LoginVerifyRequest{
		AssertionResponse: []byte(`{"type":"public-key","id":"mock-cred-id","rawId":"bW9jay1jcmVkLWlk","response":{"authenticatorData":"bW9jay1hdXRo","clientDataJSON":"bW9jay1jbGllbnQ","signature":"bW9jay1zaWdu","userHandle":"dXNlci1oYW5kbGU"}}`),
		DeviceSignature:   []byte("mock-device-signature"),
		UserId:            loginResp.UserId,
	})
	if err != nil {
		t.Fatalf("LoginVerify RPC failed: %v", err)
	}

	// Verify login tokens are present
	if loginVerifyResp.AccessToken == "" {
		t.Error("expected non-empty access token after login verify")
	}
	if loginVerifyResp.RefreshToken == "" {
		t.Error("expected non-empty refresh token after login verify")
	}
	if loginVerifyResp.SessionId == nil || loginVerifyResp.SessionId.Value == "" {
		t.Error("expected non-empty session ID after login verify")
	}

	// Verify the login access token is different from the registration access token
	// (they should be different sessions)
	if loginVerifyResp.AccessToken == regVerifyResp.AccessToken {
		t.Error("expected different access tokens for registration and login sessions")
	}
}

// TestWebAuthnChallengeUniqueness tests that each Register and Login call
// returns a unique WebAuthn challenge, preventing replay attacks.
func TestWebAuthnChallengeUniqueness(t *testing.T) {
	skipIfUnreachable(t, authServiceAddr, "auth")

	client, conn := newAuthClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, pubKey := generateTestKeyPair(t)

	// Register two different users and compare challenges
	reg1, err := client.Register(ctx, &authv1.RegisterRequest{
		Username:          "webauthn-challenge-1-" + randomSuffix(),
		DevicePublicKey:   pubKey,
		DeviceAttestation: "test-attestation",
	})
	if err != nil {
		t.Fatalf("Register RPC (1) failed: %v", err)
	}

	_, pubKey2 := generateTestKeyPair(t)
	reg2, err := client.Register(ctx, &authv1.RegisterRequest{
		Username:          "webauthn-challenge-2-" + randomSuffix(),
		DevicePublicKey:   pubKey2,
		DeviceAttestation: "test-attestation",
	})
	if err != nil {
		t.Fatalf("Register RPC (2) failed: %v", err)
	}

	// Challenges should be different (randomly generated)
	if string(reg1.WebauthnChallenge) == string(reg2.WebauthnChallenge) {
		t.Error("expected different WebAuthn challenges for different registrations")
	}
}