package service

import (
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"
)

func TestGenerateRegistrationChallenge(t *testing.T) {
	challenge, err := generateRegistrationChallenge("test-user-id")
	if err != nil {
		t.Fatalf("generateRegistrationChallenge failed: %v", err)
	}

	if len(challenge) != 32 {
		t.Errorf("expected 32-byte challenge, got %d bytes", len(challenge))
	}
}

func TestGenerateRegistrationChallengeUniqueness(t *testing.T) {
	ch1, _ := generateRegistrationChallenge("user-1")
	ch2, _ := generateRegistrationChallenge("user-1")

	// Challenges should be different (extremely unlikely to collide)
	same := true
	for i := range ch1 {
		if ch1[i] != ch2[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("expected different challenges for consecutive calls")
	}
}

func TestGenerateAssertionChallenge(t *testing.T) {
	challenge, err := generateAssertionChallenge("test-user-id")
	if err != nil {
		t.Fatalf("generateAssertionChallenge failed: %v", err)
	}

	if len(challenge) != 32 {
		t.Errorf("expected 32-byte challenge, got %d bytes", len(challenge))
	}
}

func TestGenerateAssertionChallengeUniqueness(t *testing.T) {
	ch1, _ := generateAssertionChallenge("user-1")
	ch2, _ := generateAssertionChallenge("user-1")

	same := true
	for i := range ch1 {
		if ch1[i] != ch2[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("expected different challenges for consecutive calls")
	}
}

func TestIsDevKey(t *testing.T) {
	tests := []struct {
		name      string
		publicKey []byte
		expected  bool
	}{
		{
			name:      "dev key prefix",
			publicKey: []byte("dev:dGVzdA=="),
			expected:  true,
		},
		{
			name:      "non-dev key",
			publicKey: []byte("regularkey"),
			expected:  false,
		},
		{
			name:      "empty key",
			publicKey: []byte(""),
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isDevKey(tt.publicKey)
			if result != tt.expected {
				t.Errorf("isDevKey(%q) = %v, want %v", tt.publicKey, result, tt.expected)
			}
		})
	}
}

func TestVerifyRegistrationCredentialDevBypass(t *testing.T) {
	// Test dev bypass with valid base64 content
	devKey := []byte("dev:dGVzdHB1YmxpY2tleQ==")
	result, err := verifyRegistrationCredentialDevBypass(devKey)
	if err != nil {
		t.Fatalf("verifyRegistrationCredentialDevBypass failed: %v", err)
	}
	if string(result.CredentialID) != "dev-credential" {
		t.Errorf("expected dev-credential ID, got %s", string(result.CredentialID))
	}
	if result.AttestationType != "none" {
		t.Errorf("expected 'none' attestation type, got %s", result.AttestationType)
	}
}

func TestVerifyAssertionCredentialDevBypass(t *testing.T) {
	result := verifyAssertionCredentialDevBypass()
	if string(result.CredentialID) != "dev-credential" {
		t.Errorf("expected dev-credential ID, got %s", string(result.CredentialID))
	}
}

func TestChallengeStore(t *testing.T) {
	store := NewMemoryChallengeStore()
	ctx := t.Context()

	// Test store and retrieve
	userID := "test-user-123"
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		t.Fatalf("failed to generate challenge: %v", err)
	}

	encoded := base64.StdEncoding.EncodeToString(challenge)
	if err := store.Put(ctx, userID, encoded, 5*time.Minute); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	retrieved, err := store.Get(ctx, userID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(retrieved)
	if err != nil {
		t.Fatalf("failed to decode retrieved challenge: %v", err)
	}

	if len(decoded) != len(challenge) {
		t.Errorf("expected %d bytes, got %d", len(challenge), len(decoded))
	}

	for i := range challenge {
		if challenge[i] != decoded[i] {
			t.Error("challenge data mismatch")
			break
		}
	}
}

func TestDeviceToCredentials(t *testing.T) {
	// Test with empty devices
	creds := deviceToCredentials(nil)
	if len(creds) != 0 {
		t.Errorf("expected 0 credentials for nil devices, got %d", len(creds))
	}
}

func BenchmarkGenerateRegistrationChallenge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = generateRegistrationChallenge("bench-user")
	}
}

func BenchmarkGenerateAssertionChallenge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = generateAssertionChallenge("bench-user")
	}
}

func BenchmarkRandomBytes(b *testing.B) {
	buf := make([]byte, 32)
	for i := 0; i < b.N; i++ {
		_, _ = rand.Read(buf)
	}
}