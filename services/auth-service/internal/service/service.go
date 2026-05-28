package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/nomados/nomados/packages/logging"
	"github.com/nomados/nomados/services/auth-service/internal/repository"
)

// webauthnUser implements the webauthn.User interface for our auth service.
type webauthnUser struct {
	id          []byte
	username    string
	displayName string
}

func (u *webauthnUser) WebAuthnID() []byte {
	return u.id
}

func (u *webauthnUser) WebAuthnName() string {
	return u.username
}

func (u *webauthnUser) WebAuthnDisplayName() string {
	return u.displayName
}

func (u *webauthnUser) WebAuthnCredentials() []webauthn.Credential {
	// Credentials are looked up per-device; this returns an empty slice
	// because we populate credentials dynamically in the verification flows.
	return nil
}

// RegistrationResult holds the result of a user registration.
type RegistrationResult struct {
	User      *repository.User
	Device    *repository.Device
	Challenge []byte
}

// AuthService provides business logic for authentication flows.
type AuthService struct {
	repo   *repository.PostgresRepository
	logger *logging.Logger
	webauthn *webauthn.WebAuthn
}

// NewAuthService creates a new AuthService.
func NewAuthService(repo *repository.PostgresRepository) *AuthService {
	logger := logging.NewLogger("auth-service", nil)

	cfg := &WebAuthnConfig{
		RPID:      envOrDefault("WEBAUTHN_RP_ID", "localhost"),
		RPOrigins: []string{envOrDefault("WEBAUTHN_ORIGIN", "http://localhost:3000")},
	}

	w, err := NewWebAuthn(cfg)
	if err != nil {
		logger.Error("failed to initialize WebAuthn, running without verification", "error", err)
		// webauthn will be nil; verification calls will fall back to dev bypass
	}

	return &AuthService{
		repo:     repo,
		logger:   logger,
		webauthn: w,
	}
}

// envOrDefault returns the value of the environment variable or the default.
func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// Register creates a new user and their first device, returning a WebAuthn challenge.
func (s *AuthService) Register(ctx context.Context, username string, devicePublicKey []byte, deviceAttestation string) (*RegistrationResult, error) {
	// Check if user already exists
	existing, err := s.repo.GetUserByUsername(ctx, username)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("user already exists: %s", username)
	}

	user, err := s.repo.CreateUser(ctx, username)
	if err != nil {
		s.logger.Error("failed to create user", "username", username, "error", err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	device, err := s.repo.CreateDevice(ctx, user.ID, devicePublicKey, deviceAttestation)
	if err != nil {
		s.logger.Error("failed to create device", "user_id", user.ID, "error", err)
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	// Generate a WebAuthn registration challenge
	challenge, err := generateRegistrationChallenge(user.ID.String())
	if err != nil {
		s.logger.Error("failed to generate challenge", "user_id", user.ID, "error", err)
		return nil, fmt.Errorf("failed to generate challenge: %w", err)
	}

	// Store the challenge for later verification
	storeChallengeForUser(user.ID.String(), challenge)

	s.logger.Info("user registered", "user_id", user.ID, "username", username)

	// Audit the registration
	_ = s.CreateAuditEntry(ctx, user.ID, "user.register", user.ID.String(), map[string]interface{}{
		"username": username,
	})

	return &RegistrationResult{
		User:      user,
		Device:    device,
		Challenge: challenge,
	}, nil
}

// VerifyRegistration marks a device as trusted after WebAuthn verification.
// For dev mode (device_public_key starts with "dev:"), verification is bypassed.
// For real WebAuthn, the credential response is verified and the credential data is stored.
func (s *AuthService) VerifyRegistration(ctx context.Context, userID uuid.UUID, credentialResponse []byte, deviceSignature []byte) error {
	// Get user devices
	devices, err := s.repo.GetDevicesByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get devices: %w", err)
	}
	if len(devices) == 0 {
		return fmt.Errorf("no devices found for user")
	}

	device := devices[0]

	if isDevKey(device.PublicKey) {
		// Dev mode bypass: skip real verification
		result, err := verifyRegistrationCredentialDevBypass(device.PublicKey)
		if err != nil {
			return fmt.Errorf("dev registration verification failed: %w", err)
		}

		// Store the dev credential data
		if err := s.repo.UpdateDeviceCredential(ctx, device.ID, result.CredentialID, result.CredentialPublicKey); err != nil {
			s.logger.Warn("failed to update dev credential data", "device_id", device.ID, "error", err)
		}

		if err := s.repo.TrustDevice(ctx, device.ID); err != nil {
			s.logger.Error("failed to trust device", "device_id", device.ID, "error", err)
			return fmt.Errorf("failed to trust device: %w", err)
		}

		s.logger.Info("registration verified (dev mode)", "user_id", userID, "device_id", device.ID)
	} else {
		// Real WebAuthn verification
		if s.webauthn == nil {
			return fmt.Errorf("WebAuthn not initialized, cannot verify real credentials")
		}

		// Look up the challenge that was stored for this user during registration
		// For now, we reconstruct it from the devicePublicKey-based challenge store
		// The challenge was generated during Register and sent back to the client
		// The client should send it back in the credential response
		// We need to retrieve it from the challenge store
		challenge, err := getChallengeForUser(userID.String())
		if err != nil {
			return fmt.Errorf("challenge not found or expired: %w", err)
		}

		// Create a webauthn.User for the verification
		webauthnUser := &webauthnUser{
			id:          userID[:],
			username:    userID.String(),
			displayName: userID.String(),
		}

		result, err := verifyRegistrationCredential(credentialResponse, challenge, s.webauthn, webauthnUser)
		if err != nil {
			return fmt.Errorf("registration verification failed: %w", err)
		}

		// Store the verified credential data
		if err := s.repo.UpdateDeviceCredential(ctx, device.ID, result.CredentialID, result.CredentialPublicKey); err != nil {
			s.logger.Error("failed to store credential data", "device_id", device.ID, "error", err)
			return fmt.Errorf("failed to store credential data: %w", err)
		}

		if err := s.repo.TrustDevice(ctx, device.ID); err != nil {
			s.logger.Error("failed to trust device", "device_id", device.ID, "error", err)
			return fmt.Errorf("failed to trust device: %w", err)
		}

		s.logger.Info("registration verified", "user_id", userID, "device_id", device.ID)
	}

	_ = s.CreateAuditEntry(ctx, userID, "user.verify_registration", device.ID.String(), map[string]interface{}{
		"device_id": device.ID.String(),
		"dev_mode":  isDevKey(device.PublicKey),
	})

	return nil
}

// Login looks up a user by their device public key and returns a challenge.
func (s *AuthService) Login(ctx context.Context, devicePublicKey []byte) (*repository.User, []byte, error) {
	device, err := s.repo.GetDeviceByPublicKey(ctx, devicePublicKey)
	if err != nil {
		s.logger.Error("device not found for login", "error", err)
		return nil, nil, fmt.Errorf("device not found")
	}

	user, err := s.repo.GetUserByID(ctx, device.UserID)
	if err != nil {
		s.logger.Error("user not found for login", "device_id", device.ID, "error", err)
		return nil, nil, fmt.Errorf("user not found")
	}

	// Update device last seen
	if err := s.repo.UpdateDeviceLastSeen(ctx, device.ID); err != nil {
		s.logger.Warn("failed to update device last_seen", "device_id", device.ID, "error", err)
	}

	// Generate assertion challenge
	challenge, err := generateAssertionChallenge(user.ID.String())
	if err != nil {
		s.logger.Error("failed to generate assertion challenge", "user_id", user.ID, "error", err)
		return nil, nil, fmt.Errorf("failed to generate challenge: %w", err)
	}

	// Store the challenge for later verification
	storeChallengeForUser(user.ID.String(), challenge)

	s.logger.Info("user login initiated", "user_id", user.ID)

	return user, challenge, nil
}

// VerifyAssertion verifies a WebAuthn assertion for login.
// For dev mode (device_public_key starts with "dev:"), verification is bypassed.
// For real WebAuthn, the assertion is verified against the stored credential.
func (s *AuthService) VerifyAssertion(ctx context.Context, userID uuid.UUID, assertionResponse []byte, deviceSignature []byte) error {
	// Get user devices
	devices, err := s.repo.GetDevicesByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get devices: %w", err)
	}
	if len(devices) == 0 {
		return fmt.Errorf("no devices found for user")
	}

	device := devices[0]

	if isDevKey(device.PublicKey) {
		// Dev mode bypass: skip real verification
		_ = verifyAssertionCredentialDevBypass()

		// Update sign count (no-op in dev mode, but good practice)
		if err := s.repo.UpdateDeviceSignCount(ctx, device.ID, 0); err != nil {
			s.logger.Warn("failed to update device sign count", "device_id", device.ID, "error", err)
		}

		s.logger.Info("assertion verified (dev mode)", "user_id", userID, "device_id", device.ID)
	} else {
		// Real WebAuthn verification
		if s.webauthn == nil {
			return fmt.Errorf("WebAuthn not initialized, cannot verify real assertions")
		}

		// Look up the stored challenge for this user
		challenge, err := getChallengeForUser(userID.String())
		if err != nil {
			return fmt.Errorf("challenge not found or expired: %w", err)
		}

		// Create a webauthn.User for verification
		webauthnUser := &webauthnUser{
			id:          userID[:],
			username:    userID.String(),
			displayName: userID.String(),
		}

		// Build stored credentials from device data
		storedCredentials := deviceToCredentials(devices)

		result, err := verifyAssertionCredential(assertionResponse, challenge, s.webauthn, webauthnUser, storedCredentials)
		if err != nil {
			return fmt.Errorf("assertion verification failed: %w", err)
		}

		// Update sign count for replay protection
		if err := s.repo.UpdateDeviceSignCount(ctx, device.ID, int64(result.SignCount)); err != nil {
			s.logger.Warn("failed to update device sign count", "device_id", device.ID, "error", err)
		}

		s.logger.Info("assertion verified", "user_id", userID, "device_id", device.ID)
	}

	return nil
}

// GetDevicesForUser returns all devices registered for a user.
func (s *AuthService) GetDevicesForUser(ctx context.Context, userID uuid.UUID) ([]*repository.Device, error) {
	devices, err := s.repo.GetDevicesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}
	return devices, nil
}

// CreateAuditEntry logs an audit event to the database.
func (s *AuthService) CreateAuditEntry(ctx context.Context, actorID uuid.UUID, action, target string, details map[string]interface{}) error {
	return s.repo.CreateAuditLog(ctx, actorID, action, target, details)
}

// deviceToCredentials converts repository devices to webauthn.Credential slice
// for use in assertion verification.
func deviceToCredentials(devices []*repository.Device) []webauthn.Credential {
	credentials := make([]webauthn.Credential, 0, len(devices))
	for _, d := range devices {
		if len(d.CredentialID) > 0 && len(d.CredentialPublicKey) > 0 {
			cred := webauthn.Credential{
				ID:        d.CredentialID,
				PublicKey:  d.CredentialPublicKey,
				Authenticator: webauthn.Authenticator{
					SignCount: uint32(d.SignCount),
				},
			}
			credentials = append(credentials, cred)
		}
	}
	return credentials
}

// challengeStore provides in-memory challenge storage for WebAuthn ceremonies.
// This is a temporary implementation; Task 2 adds Redis-backed persistence.
type challengeEntry struct {
	challenge []byte
}

var globalChallengeStore = struct {
	entries map[string]*challengeEntry
}{
	entries: make(map[string]*challengeEntry),
}

// storeChallengeForUser stores a challenge for a user (for later verification).
func storeChallengeForUser(userID string, challenge []byte) {
	globalChallengeStore.entries[userID] = &challengeEntry{challenge: challenge}
}

// getChallengeForUser retrieves a stored challenge for a user.
func getChallengeForUser(userID string) ([]byte, error) {
	entry, ok := globalChallengeStore.entries[userID]
	if !ok {
		return nil, fmt.Errorf("challenge not found for user %s", userID)
	}
	// Delete the challenge after retrieval (one-time use)
	delete(globalChallengeStore.entries, userID)
	return entry.challenge, nil
}

// isDevPublicKey checks if a device public key is a dev-mode key (starts with "dev:").
// This is exported for use in the handler layer.
func isDevPublicKey(publicKey []byte) bool {
	return strings.HasPrefix(string(publicKey), "dev:")
}

// decodeDevPublicKey decodes a dev-mode public key (base64 after "dev:" prefix).
func decodeDevPublicKey(publicKey []byte) ([]byte, error) {
	encoded := strings.TrimPrefix(string(publicKey), "dev:")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode dev public key: %w", err)
	}
	return decoded, nil
}