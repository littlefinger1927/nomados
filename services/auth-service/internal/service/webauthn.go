package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

const challengeTTL = 5 * time.Minute

// generateRegistrationChallenge creates a random 32-byte challenge for WebAuthn registration,
// stores it in the ChallengeStore under the given userID, and returns the challenge bytes.
func generateRegistrationChallenge(ctx context.Context, store ChallengeStore, userID string) ([]byte, error) {
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return nil, fmt.Errorf("failed to generate challenge: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(challenge)
	if err := store.Put(ctx, fmt.Sprintf("reg:%s", userID), encoded, challengeTTL); err != nil {
		return nil, fmt.Errorf("failed to store challenge: %w", err)
	}
	return challenge, nil
}

// verifyRegistrationCredential verifies a WebAuthn registration credential against a challenge.
// This is a stub — always returns nil (success) for now.
func verifyRegistrationCredential(credential []byte, challenge []byte) error {
	// TODO: Implement real WebAuthn credential verification
	return nil
}

// generateAssertionChallenge creates a random 32-byte challenge for WebAuthn login (assertion),
// stores it in the ChallengeStore under the given userID, and returns the challenge bytes.
func generateAssertionChallenge(ctx context.Context, store ChallengeStore, userID string) ([]byte, error) {
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return nil, fmt.Errorf("failed to generate assertion challenge: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(challenge)
	if err := store.Put(ctx, fmt.Sprintf("auth:%s", userID), encoded, challengeTTL); err != nil {
		return nil, fmt.Errorf("failed to store assertion challenge: %w", err)
	}
	return challenge, nil
}

// verifyAssertionCredential verifies a WebAuthn assertion credential against a challenge.
// This is a stub — always returns nil (success) for now.
func verifyAssertionCredential(assertion []byte, challenge []byte) error {
	// TODO: Implement real WebAuthn assertion verification
	return nil
}