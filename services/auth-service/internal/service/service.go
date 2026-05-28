package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nomados/nomados/packages/logging"
	"github.com/nomados/nomados/services/auth-service/internal/repository"
)

// RegistrationResult holds the result of a user registration.
type RegistrationResult struct {
	User         *repository.User
	Device       *repository.Device
	Challenge    []byte
}

// AuthService provides business logic for authentication flows.
type AuthService struct {
	repo           *repository.PostgresRepository
	logger         *logging.Logger
	challengeStore ChallengeStore
}

// NewAuthService creates a new AuthService.
func NewAuthService(repo *repository.PostgresRepository, challengeStore ChallengeStore) *AuthService {
	return &AuthService{
		repo:           repo,
		logger:         logging.NewLogger("auth-service", nil),
		challengeStore: challengeStore,
	}
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
	challenge, err := generateRegistrationChallenge(ctx, s.challengeStore, user.ID.String())
	if err != nil {
		s.logger.Error("failed to generate challenge", "user_id", user.ID, "error", err)
		return nil, fmt.Errorf("failed to generate challenge: %w", err)
	}

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
	challenge, err := generateAssertionChallenge(ctx, s.challengeStore, user.ID.String())
	if err != nil {
		s.logger.Error("failed to generate assertion challenge", "user_id", user.ID, "error", err)
		return nil, nil, fmt.Errorf("failed to generate challenge: %w", err)
	}

	s.logger.Info("user login initiated", "user_id", user.ID)

	return user, challenge, nil
}

// GetDevicesForUser returns all devices registered for a user.
func (s *AuthService) GetDevicesForUser(ctx context.Context, userID uuid.UUID) ([]*repository.Device, error) {
	devices, err := s.repo.GetDevicesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}
	return devices, nil
}

// VerifyRegistration marks a device as trusted after WebAuthn verification.
// Currently a stub — real WebAuthn verification will be added in a future iteration.
func (s *AuthService) VerifyRegistration(ctx context.Context, userID uuid.UUID, credentialResponse []byte, deviceSignature []byte) error {
	// Get user devices
	devices, err := s.repo.GetDevicesByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get devices: %w", err)
	}
	if len(devices) == 0 {
		return fmt.Errorf("no devices found for user")
	}

	// Mark the first device as trusted (stub behavior)
	device := devices[0]
	if err := s.repo.TrustDevice(ctx, device.ID); err != nil {
		s.logger.Error("failed to trust device", "device_id", device.ID, "error", err)
		return fmt.Errorf("failed to trust device: %w", err)
	}

	s.logger.Info("registration verified", "user_id", userID, "device_id", device.ID)

	_ = s.CreateAuditEntry(ctx, userID, "user.verify_registration", device.ID.String(), map[string]interface{}{
		"device_id": device.ID.String(),
	})

	return nil
}

// CreateAuditEntry logs an audit event to the database.
func (s *AuthService) CreateAuditEntry(ctx context.Context, actorID uuid.UUID, action, target string, details map[string]interface{}) error {
	return s.repo.CreateAuditLog(ctx, actorID, action, target, details)
}