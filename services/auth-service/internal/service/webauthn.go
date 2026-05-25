package service

import (
	"crypto/rand"
	"fmt"
)

// generateRegistrationChallenge creates a random 32-byte challenge for WebAuthn registration.
// This is a stub — real WebAuthn challenge generation will be implemented in a future iteration.
func generateRegistrationChallenge(userID string) ([]byte, error) {
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return nil, fmt.Errorf("failed to generate challenge: %w", err)
	}
	return challenge, nil
}

// verifyRegistrationCredential verifies a WebAuthn registration credential against a challenge.
// This is a stub — always returns nil (success) for now.
func verifyRegistrationCredential(credential []byte, challenge []byte) error {
	// TODO: Implement real WebAuthn credential verification
	return nil
}

// generateAssertionChallenge creates a random 32-byte challenge for WebAuthn login (assertion).
// This is a stub — real WebAuthn challenge generation will be implemented in a future iteration.
func generateAssertionChallenge(userID string) ([]byte, error) {
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return nil, fmt.Errorf("failed to generate assertion challenge: %w", err)
	}
	return challenge, nil
}

// verifyAssertionCredential verifies a WebAuthn assertion credential against a challenge.
// This is a stub — always returns nil (success) for now.
func verifyAssertionCredential(assertion []byte, challenge []byte) error {
	// TODO: Implement real WebAuthn assertion verification
	return nil
}