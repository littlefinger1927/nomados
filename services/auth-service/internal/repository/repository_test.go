package repository

import (
	"context"
	"os"
	"testing"

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

func TestCreateUser(t *testing.T) {
	pool := getTestPool(t)
	if pool == nil {
		return
	}

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	username := "testuser_" + uuid.New().String()[:8]
	user, err := repo.CreateUser(ctx, username)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if user.ID == uuid.Nil {
		t.Error("expected non-nil user ID")
	}
	if user.Username != username {
		t.Errorf("expected username %s, got %s", username, user.Username)
	}
	if user.Status != "active" {
		t.Errorf("expected status active, got %s", user.Status)
	}
}

func TestGetUserByID(t *testing.T) {
	pool := getTestPool(t)
	if pool == nil {
		return
	}

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	username := "testuser_" + uuid.New().String()[:8]
	created, err := repo.CreateUser(ctx, username)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	found, err := repo.GetUserByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}

	if found.Username != username {
		t.Errorf("expected username %s, got %s", username, found.Username)
	}
}

func TestGetUserByUsername(t *testing.T) {
	pool := getTestPool(t)
	if pool == nil {
		return
	}

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	username := "testuser_" + uuid.New().String()[:8]
	created, err := repo.CreateUser(ctx, username)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	found, err := repo.GetUserByUsername(ctx, username)
	if err != nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}

	if found.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, found.ID)
	}
}

func TestCreateDevice(t *testing.T) {
	pool := getTestPool(t)
	if pool == nil {
		return
	}

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	username := "testuser_" + uuid.New().String()[:8]
	user, err := repo.CreateUser(ctx, username)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	publicKey := []byte("test-public-key-data")
	attestation := "test-attestation"

	device, err := repo.CreateDevice(ctx, user.ID, publicKey, attestation)
	if err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	if device.ID == uuid.Nil {
		t.Error("expected non-nil device ID")
	}
	if device.UserID != user.ID {
		t.Errorf("expected user_id %s, got %s", user.ID, device.UserID)
	}
	if device.Trusted {
		t.Error("expected device to not be trusted initially")
	}
}

func TestGetDevicesByUserID(t *testing.T) {
	pool := getTestPool(t)
	if pool == nil {
		return
	}

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	username := "testuser_" + uuid.New().String()[:8]
	user, err := repo.CreateUser(ctx, username)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	publicKey1 := []byte("test-public-key-1")
	publicKey2 := []byte("test-public-key-2")

	_, err = repo.CreateDevice(ctx, user.ID, publicKey1, "attestation-1")
	if err != nil {
		t.Fatalf("CreateDevice 1 failed: %v", err)
	}
	_, err = repo.CreateDevice(ctx, user.ID, publicKey2, "attestation-2")
	if err != nil {
		t.Fatalf("CreateDevice 2 failed: %v", err)
	}

	devices, err := repo.GetDevicesByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetDevicesByUserID failed: %v", err)
	}

	if len(devices) < 2 {
		t.Errorf("expected at least 2 devices, got %d", len(devices))
	}
}

func TestUpdateDeviceLastSeen(t *testing.T) {
	pool := getTestPool(t)
	if pool == nil {
		return
	}

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	username := "testuser_" + uuid.New().String()[:8]
	user, err := repo.CreateUser(ctx, username)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	device, err := repo.CreateDevice(ctx, user.ID, []byte("test-pk"), "test-att")
	if err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	err = repo.UpdateDeviceLastSeen(ctx, device.ID)
	if err != nil {
		t.Fatalf("UpdateDeviceLastSeen failed: %v", err)
	}

	updated, err := repo.GetDeviceByID(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetDeviceByID failed: %v", err)
	}

	if updated.LastSeen == nil {
		t.Error("expected last_seen to be set")
	}
}

func TestCreateAuditLog(t *testing.T) {
	pool := getTestPool(t)
	if pool == nil {
		return
	}

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	username := "testuser_" + uuid.New().String()[:8]
	user, err := repo.CreateUser(ctx, username)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	err = repo.CreateAuditLog(ctx, user.ID, "user.login", user.ID.String(), map[string]interface{}{
		"ip": "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("CreateAuditLog failed: %v", err)
	}
}