package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func getTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://nomados:nomados_dev@localhost:5432/nomados?sslmode=disable"
	}

	ctx := context.Background()
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		t.Skipf("database not available: %v", err)
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Skipf("database not available: %v", err)
		return nil
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("database not available: %v", err)
		return nil
	}

	t.Cleanup(func() { pool.Close() })
	return pool
}

// createTestUser creates a user for testing (assumes auth service's users table exists).
func createTestUser(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	username := "session_test_" + uuid.New().String()[:8]
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (username, status) VALUES ($1, 'active') RETURNING id`,
		username,
	).Scan(&id)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return id
}

// createTestDevice creates a device for testing.
func createTestDevice(t *testing.T, pool *pgxpool.Pool, userID string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO devices (user_id, public_key, attestation, trusted) VALUES ($1, $2, $3, false) RETURNING id`,
		userID, []byte("test-public-key"), "test-attestation",
	).Scan(&id)
	if err != nil {
		t.Fatalf("failed to create test device: %v", err)
	}
	return id
}

func TestCreateSession(t *testing.T) {
	pool := getTestPool(t)
	if pool == nil {
		return
	}

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	userID := createTestUser(t, pool)
	deviceID := createTestDevice(t, pool, userID)

	expiresAt := time.Now().Add(24 * time.Hour)
	session, err := repo.CreateSession(ctx, userID, deviceID, "ip-hash-123", 0, expiresAt)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if session.ID == uuid.Nil {
		t.Error("expected non-nil session ID")
	}
	if session.UserID.String() != userID {
		t.Errorf("expected user_id %s, got %s", userID, session.UserID.String())
	}
	if session.DeviceID.String() != deviceID {
		t.Errorf("expected device_id %s, got %s", deviceID, session.DeviceID.String())
	}
	if session.IPHash != "ip-hash-123" {
		t.Errorf("expected ip_hash ip-hash-123, got %s", session.IPHash)
	}
	if session.Revoked {
		t.Error("expected session to not be revoked initially")
	}
}

func TestGetSessionByID(t *testing.T) {
	pool := getTestPool(t)
	if pool == nil {
		return
	}

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	userID := createTestUser(t, pool)
	deviceID := createTestDevice(t, pool, userID)

	expiresAt := time.Now().Add(24 * time.Hour)
	created, err := repo.CreateSession(ctx, userID, deviceID, "", 0, expiresAt)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	found, err := repo.GetSessionByID(ctx, created.ID.String())
	if err != nil {
		t.Fatalf("GetSessionByID failed: %v", err)
	}

	if found.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, found.ID)
	}
}

func TestGetSessionsByUserID(t *testing.T) {
	pool := getTestPool(t)
	if pool == nil {
		return
	}

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	userID := createTestUser(t, pool)
	deviceID := createTestDevice(t, pool, userID)

	expiresAt := time.Now().Add(24 * time.Hour)
	_, err := repo.CreateSession(ctx, userID, deviceID, "", 0, expiresAt)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	_, err = repo.CreateSession(ctx, userID, deviceID, "", 0, expiresAt)
	if err != nil {
		t.Fatalf("CreateSession 2 failed: %v", err)
	}

	sessions, err := repo.GetSessionsByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("GetSessionsByUserID failed: %v", err)
	}

	if len(sessions) < 2 {
		t.Errorf("expected at least 2 sessions, got %d", len(sessions))
	}
}

func TestValidateSession(t *testing.T) {
	pool := getTestPool(t)
	if pool == nil {
		return
	}

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	userID := createTestUser(t, pool)
	deviceID := createTestDevice(t, pool, userID)

	expiresAt := time.Now().Add(24 * time.Hour)
	created, err := repo.CreateSession(ctx, userID, deviceID, "", 0, expiresAt)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Valid session should be found
	validated, err := repo.ValidateSession(ctx, created.ID.String())
	if err != nil {
		t.Fatalf("ValidateSession failed: %v", err)
	}
	if validated.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, validated.ID)
	}
}

func TestValidateSessionExpired(t *testing.T) {
	pool := getTestPool(t)
	if pool == nil {
		return
	}

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	userID := createTestUser(t, pool)
	deviceID := createTestDevice(t, pool, userID)

	// Create a session that is already expired
	expiresAt := time.Now().Add(-1 * time.Hour)
	created, err := repo.CreateSession(ctx, userID, deviceID, "", 0, expiresAt)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Expired session should not be found
	_, err = repo.ValidateSession(ctx, created.ID.String())
	if err == nil {
		t.Error("expected error for expired session, got nil")
	}
}

func TestRevokeSession(t *testing.T) {
	pool := getTestPool(t)
	if pool == nil {
		return
	}

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	userID := createTestUser(t, pool)
	deviceID := createTestDevice(t, pool, userID)

	expiresAt := time.Now().Add(24 * time.Hour)
	created, err := repo.CreateSession(ctx, userID, deviceID, "", 0, expiresAt)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	err = repo.RevokeSession(ctx, created.ID.String())
	if err != nil {
		t.Fatalf("RevokeSession failed: %v", err)
	}

	// Revoked session should not be found by ValidateSession
	_, err = repo.ValidateSession(ctx, created.ID.String())
	if err == nil {
		t.Error("expected error for revoked session, got nil")
	}

	// But should still be found by GetSessionByID
	found, err := repo.GetSessionByID(ctx, created.ID.String())
	if err != nil {
		t.Fatalf("GetSessionByID failed: %v", err)
	}
	if !found.Revoked {
		t.Error("expected session to be revoked")
	}
}

func TestRevokeSessionsByUserID(t *testing.T) {
	pool := getTestPool(t)
	if pool == nil {
		return
	}

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	userID := createTestUser(t, pool)
	deviceID := createTestDevice(t, pool, userID)

	expiresAt := time.Now().Add(24 * time.Hour)
	_, err := repo.CreateSession(ctx, userID, deviceID, "", 0, expiresAt)
	if err != nil {
		t.Fatalf("CreateSession 1 failed: %v", err)
	}
	_, err = repo.CreateSession(ctx, userID, deviceID, "", 0, expiresAt)
	if err != nil {
		t.Fatalf("CreateSession 2 failed: %v", err)
	}

	err = repo.RevokeSessionsByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("RevokeSessionsByUserID failed: %v", err)
	}

	sessions, err := repo.GetSessionsByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("GetSessionsByUserID failed: %v", err)
	}

	for _, s := range sessions {
		if !s.Revoked {
			t.Error("expected all sessions to be revoked")
		}
	}
}

func TestUpdateRiskScore(t *testing.T) {
	pool := getTestPool(t)
	if pool == nil {
		return
	}

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	userID := createTestUser(t, pool)
	deviceID := createTestDevice(t, pool, userID)

	expiresAt := time.Now().Add(24 * time.Hour)
	created, err := repo.CreateSession(ctx, userID, deviceID, "", 0, expiresAt)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	err = repo.UpdateRiskScore(ctx, created.ID.String(), 75)
	if err != nil {
		t.Fatalf("UpdateRiskScore failed: %v", err)
	}

	found, err := repo.GetSessionByID(ctx, created.ID.String())
	if err != nil {
		t.Fatalf("GetSessionByID failed: %v", err)
	}
	if found.RiskScore != 75 {
		t.Errorf("expected risk_score 75, got %d", found.RiskScore)
	}
}