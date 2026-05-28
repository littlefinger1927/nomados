package service

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// WebAuthnConfig holds configuration for the WebAuthn Relying Party.
type WebAuthnConfig struct {
	RPID   string
	RPOrigins []string
}

// DefaultWebAuthnConfig returns a WebAuthnConfig with sensible defaults for development.
func DefaultWebAuthnConfig() *WebAuthnConfig {
	return &WebAuthnConfig{
		RPID:   "localhost",
		RPOrigins: []string{"http://localhost:3000"},
	}
}

// NewWebAuthn creates a new WebAuthn instance from the given config.
func NewWebAuthn(cfg *WebAuthnConfig) (*webauthn.WebAuthn, error) {
	wcfg := &webauthn.Config{
		RPID:         cfg.RPID,
		RPDisplayName: "NomadOS",
		RPOrigins:     cfg.RPOrigins,
	}
	return webauthn.New(wcfg)
}

// generateRegistrationChallenge creates a random 32-byte challenge for WebAuthn registration.
func generateRegistrationChallenge(userID string) ([]byte, error) {
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return nil, fmt.Errorf("failed to generate challenge: %w", err)
	}
	return challenge, nil
}

// generateAssertionChallenge creates a random 32-byte challenge for WebAuthn login (assertion).
func generateAssertionChallenge(userID string) ([]byte, error) {
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return nil, fmt.Errorf("failed to generate assertion challenge: %w", err)
	}
	return challenge, nil
}

// RegistrationVerificationResult holds the extracted credential data from a verified registration.
type RegistrationVerificationResult struct {
	CredentialID        []byte
	CredentialPublicKey []byte
	AttestationType    string
	SignCount          uint32
}

// verifyRegistrationCredential verifies a WebAuthn registration credential against a challenge.
// The credentialBytes should be the JSON-encoded CredentialCreationResponse from the client.
// The challenge is the base64url-encoded challenge that was sent to the client.
// If the devicePublicKey starts with "dev:" prefix, the verification is bypassed (dev mode).
func verifyRegistrationCredential(credentialBytes []byte, challenge []byte, w *webauthn.WebAuthn, user webauthn.User) (*RegistrationVerificationResult, error) {
	// Parse the credential creation response from raw bytes
	parsedResponse, err := protocol.ParseCredentialCreationResponseBytes(credentialBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credential creation response: %w", err)
	}

	// Build session data from the challenge we generated
	challengeStr := base64.RawURLEncoding.EncodeToString(challenge)
	sessionData := webauthn.SessionData{
		Challenge:        challengeStr,
		RelyingPartyID:   w.Config.RPID,
		UserID:           user.WebAuthnID(),
		UserVerification: protocol.VerificationDiscouraged,
	}

	// Verify the credential using the go-webauthn library
	credential, err := w.CreateCredential(user, sessionData, parsedResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to verify registration credential: %w", err)
	}

	return &RegistrationVerificationResult{
		CredentialID:        credential.ID,
		CredentialPublicKey: credential.PublicKey,
		AttestationType:    credential.AttestationType,
		SignCount:          credential.Authenticator.SignCount,
	}, nil
}

// verifyRegistrationCredentialDevBypass skips WebAuthn verification in dev mode.
// The devicePublicKey with "dev:" prefix contains a base64-encoded public key.
func verifyRegistrationCredentialDevBypass(devicePublicKey []byte) (*RegistrationVerificationResult, error) {
	// Decode the base64 public key after stripping "dev:" prefix
	encoded := strings.TrimPrefix(string(devicePublicKey), "dev:")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode dev public key: %w", err)
	}

	return &RegistrationVerificationResult{
		CredentialID:        []byte("dev-credential"),
		CredentialPublicKey: decoded,
		AttestationType:    "none",
		SignCount:          0,
	}, nil
}

// AssertionVerificationResult holds the result of a verified assertion.
type AssertionVerificationResult struct {
	CredentialID []byte
	SignCount   uint32
}

// verifyAssertionCredential verifies a WebAuthn assertion credential against a stored credential.
// The assertionBytes should be the JSON-encoded CredentialAssertionResponse from the client.
// The challenge is the base64url-encoded challenge that was sent to the client.
// If the devicePublicKey starts with "dev:" prefix, the verification is bypassed (dev mode).
func verifyAssertionCredential(assertionBytes []byte, challenge []byte, w *webauthn.WebAuthn, user webauthn.User, storedCredentials []webauthn.Credential) (*AssertionVerificationResult, error) {
	// Parse the assertion response from raw bytes
	parsedResponse, err := protocol.ParseCredentialRequestResponseBytes(assertionBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse assertion response: %w", err)
	}

	// Build session data from the challenge we generated
	challengeStr := base64.RawURLEncoding.EncodeToString(challenge)
	sessionData := webauthn.SessionData{
		Challenge:            challengeStr,
		RelyingPartyID:       w.Config.RPID,
		UserID:               user.WebAuthnID(),
		UserVerification:     protocol.VerificationDiscouraged,
		AllowedCredentialIDs: getCredentialIDs(storedCredentials),
	}

	// Validate the assertion using the go-webauthn library
	credential, err := w.ValidateLogin(user, sessionData, parsedResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to verify assertion credential: %w", err)
	}

	return &AssertionVerificationResult{
		CredentialID: credential.ID,
		SignCount:   credential.Authenticator.SignCount,
	}, nil
}

// verifyAssertionCredentialDevBypass skips WebAuthn verification in dev mode.
func verifyAssertionCredentialDevBypass() *AssertionVerificationResult {
	return &AssertionVerificationResult{
		CredentialID: []byte("dev-credential"),
		SignCount:    0,
	}
}

// isDevKey checks if a device public key starts with the "dev:" prefix.
func isDevKey(publicKey []byte) bool {
	return strings.HasPrefix(string(publicKey), "dev:")
}

// getCredentialIDs extracts credential IDs from a slice of webauthn.Credential.
func getCredentialIDs(credentials []webauthn.Credential) [][]byte {
	ids := make([][]byte, len(credentials))
	for i, cred := range credentials {
		ids[i] = cred.ID
	}
	return ids
}