package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	authsdk "github.com/nomados/nomados/packages/auth-sdk"
	"github.com/nomados/nomados/packages/logging"
	"github.com/nomados/nomados/services/session-service/internal/nats"
	"github.com/nomados/nomados/services/session-service/internal/repository"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
	sessionTTL      = 24 * time.Hour
)

// SessionResponse holds the result of session creation or refresh.
type SessionResponse struct {
	SessionID    string
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// SessionClaims holds the validated session claims.
type SessionClaims struct {
	UserID    string
	SessionID string
	DeviceID  string
}

// SessionService provides business logic for session lifecycle.
type SessionService struct {
	repo     *repository.PostgresRepository
	logger   *logging.Logger
	pub      *nats.Publisher
	validator *authsdk.TokenValidator
}

// NewSessionService creates a new SessionService.
func NewSessionService(repo *repository.PostgresRepository, pub *nats.Publisher, signingSecret string) *SessionService {
	return &SessionService{
		repo:      repo,
		logger:    logging.NewLogger("session-service", nil),
		pub:       pub,
		validator: authsdk.NewTokenValidator(signingSecret),
	}
}

// CreateSession creates a new session in the database and generates tokens.
func (s *SessionService) CreateSession(ctx context.Context, userID, deviceID, ipHash string, riskScore int) (*SessionResponse, error) {
	expiresAt := time.Now().Add(sessionTTL)

	session, err := s.repo.CreateSession(ctx, userID, deviceID, ipHash, riskScore, expiresAt)
	if err != nil {
		s.logger.Error("failed to create session", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	accessToken, err := s.validator.GenerateAccessToken(userID, session.ID.String(), deviceID, accessTokenTTL)
	if err != nil {
		s.logger.Error("failed to generate access token", "session_id", session.ID, "error", err)
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.validator.GenerateAccessToken(userID, session.ID.String(), deviceID, refreshTokenTTL)
	if err != nil {
		s.logger.Error("failed to generate refresh token", "session_id", session.ID, "error", err)
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	s.logger.Info("session created", "session_id", session.ID, "user_id", userID, "device_id", deviceID)

	return &SessionResponse{
		SessionID:    session.ID.String(),
		AccessToken:   accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:     expiresAt,
	}, nil
}

// ValidateSession validates an access token and checks session/device binding.
func (s *SessionService) ValidateSession(ctx context.Context, accessToken string, devicePublicKey []byte) (*SessionClaims, error) {
	claims, err := s.validator.ValidateAccessToken(accessToken)
	if err != nil {
		s.logger.Error("access token validation failed", "error", err)
		return nil, fmt.Errorf("invalid access token: %w", err)
	}

	// Check that session is not revoked or expired
	_, err = s.repo.ValidateSession(ctx, claims.SessionID)
	if err != nil {
		s.logger.Error("session validation failed", "session_id", claims.SessionID, "error", err)
		return nil, fmt.Errorf("session invalid or expired: %w", err)
	}

	// Device binding: verify the device public key matches the session's device
	// In a full implementation, this would look up the device by ID and compare public keys.
	// For now, we validate that the token claims contain a device ID.
	if claims.DeviceID == "" {
		return nil, fmt.Errorf("token missing device binding")
	}

	s.logger.Info("session validated", "session_id", claims.SessionID, "user_id", claims.UserID, "device_id", claims.DeviceID)

	return &SessionClaims{
		UserID:    claims.UserID,
		SessionID: claims.SessionID,
		DeviceID:  claims.DeviceID,
	}, nil
}

// RefreshSession validates a refresh token, creates a new session, and revokes the old one.
func (s *SessionService) RefreshSession(ctx context.Context, refreshToken string, devicePublicKey []byte) (*SessionResponse, error) {
	claims, err := s.validator.ValidateAccessToken(refreshToken)
	if err != nil {
		s.logger.Error("refresh token validation failed", "error", err)
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Verify the old session is still valid
	_, err = s.repo.ValidateSession(ctx, claims.SessionID)
	if err != nil {
		s.logger.Error("old session not valid for refresh", "session_id", claims.SessionID, "error", err)
		return nil, fmt.Errorf("session not valid for refresh: %w", err)
	}

	// Create a new session
	expiresAt := time.Now().Add(sessionTTL)
	session, err := s.repo.CreateSession(ctx, claims.UserID, claims.DeviceID, "", 0, expiresAt)
	if err != nil {
		s.logger.Error("failed to create refreshed session", "user_id", claims.UserID, "error", err)
		return nil, fmt.Errorf("failed to create refreshed session: %w", err)
	}

	// Revoke the old session
	if err := s.repo.RevokeSession(ctx, claims.SessionID); err != nil {
		s.logger.Warn("failed to revoke old session during refresh", "session_id", claims.SessionID, "error", err)
		// Non-fatal: the new session is still valid
	}

	// Publish revocation event for the old session
	if s.pub != nil {
		if pubErr := s.pub.PublishSessionRevoked(ctx, claims.SessionID, claims.UserID); pubErr != nil {
			s.logger.Warn("failed to publish session revoked event", "session_id", claims.SessionID, "error", pubErr)
		}
	}

	accessToken, err := s.validator.GenerateAccessToken(claims.UserID, session.ID.String(), claims.DeviceID, accessTokenTTL)
	if err != nil {
		s.logger.Error("failed to generate access token on refresh", "session_id", session.ID, "error", err)
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := s.validator.GenerateAccessToken(claims.UserID, session.ID.String(), claims.DeviceID, refreshTokenTTL)
	if err != nil {
		s.logger.Error("failed to generate refresh token on refresh", "session_id", session.ID, "error", err)
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	s.logger.Info("session refreshed", "old_session_id", claims.SessionID, "new_session_id", session.ID, "user_id", claims.UserID)

	return &SessionResponse{
		SessionID:    session.ID.String(),
		AccessToken:   accessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:     expiresAt,
	}, nil
}

// RevokeSession revokes a session and publishes a revocation event.
func (s *SessionService) RevokeSession(ctx context.Context, sessionID string) error {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		s.logger.Error("failed to get session for revocation", "session_id", sessionID, "error", err)
		return fmt.Errorf("session not found: %w", err)
	}

	if err := s.repo.RevokeSession(ctx, sessionID); err != nil {
		s.logger.Error("failed to revoke session", "session_id", sessionID, "error", err)
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	if s.pub != nil {
		if pubErr := s.pub.PublishSessionRevoked(ctx, sessionID, session.UserID.String()); pubErr != nil {
			s.logger.Warn("failed to publish session revoked event", "session_id", sessionID, "error", pubErr)
		}
	}

	s.logger.Info("session revoked", "session_id", sessionID, "user_id", session.UserID)
	return nil
}

// RevokeAllUserSessions revokes all sessions for a user and publishes revocation events.
func (s *SessionService) RevokeAllUserSessions(ctx context.Context, userID string) error {
	sessions, err := s.repo.GetSessionsByUserID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get sessions for revocation", "user_id", userID, "error", err)
		return fmt.Errorf("failed to get user sessions: %w", err)
	}

	if err := s.repo.RevokeSessionsByUserID(ctx, userID); err != nil {
		s.logger.Error("failed to revoke all user sessions", "user_id", userID, "error", err)
		return fmt.Errorf("failed to revoke user sessions: %w", err)
	}

	// Publish individual revocation events for each active session
	if s.pub != nil {
		for _, session := range sessions {
			if !session.Revoked {
				if pubErr := s.pub.PublishSessionRevoked(ctx, session.ID.String(), session.UserID.String()); pubErr != nil {
					s.logger.Warn("failed to publish session revoked event", "session_id", session.ID, "error", pubErr)
				}
			}
		}
		if pubErr := s.pub.PublishAllSessionsRevoked(ctx, userID); pubErr != nil {
			s.logger.Warn("failed to publish all sessions revoked event", "user_id", userID, "error", pubErr)
		}
	}

	s.logger.Info("all user sessions revoked", "user_id", userID, "count", len(sessions))
	return nil
}

// UpdateRiskScore updates the risk score of a session.
func (s *SessionService) UpdateRiskScore(ctx context.Context, sessionID string, riskScore int) error {
	if err := s.repo.UpdateRiskScore(ctx, sessionID, riskScore); err != nil {
		s.logger.Error("failed to update risk score", "session_id", sessionID, "error", err)
		return fmt.Errorf("failed to update risk score: %w", err)
	}
	s.logger.Info("session risk score updated", "session_id", sessionID, "risk_score", riskScore)
	return nil
}

// parseUUID is a helper to parse UUID strings.
func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}