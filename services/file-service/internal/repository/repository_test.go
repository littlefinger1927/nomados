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

func TestFileMetadataStruct(t *testing.T) {
	id := uuid.New()
	wsID := uuid.New()
	now := time.Now()

	f := FileMetadata{
		ID:            id,
		WorkspaceID:   wsID,
		Filename:      "test.pdf",
		Size:          1024,
		ContentType:   "application/pdf",
		EncryptedPath: "ws-123/file-456",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if f.ID != id {
		t.Errorf("expected ID %s, got %s", id, f.ID)
	}
	if f.WorkspaceID != wsID {
		t.Errorf("expected WorkspaceID %s, got %s", wsID, f.WorkspaceID)
	}
	if f.Filename != "test.pdf" {
		t.Errorf("expected Filename test.pdf, got %s", f.Filename)
	}
	if f.Size != 1024 {
		t.Errorf("expected Size 1024, got %d", f.Size)
	}
	if f.ContentType != "application/pdf" {
		t.Errorf("expected ContentType application/pdf, got %s", f.ContentType)
	}
	if f.EncryptedPath != "ws-123/file-456" {
		t.Errorf("expected EncryptedPath ws-123/file-456, got %s", f.EncryptedPath)
	}
}

// createTestWorkspace inserts a user and workspace row so file FK constraints are satisfied.
func createTestWorkspace(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, username, status) VALUES ($1, $2, 'active')`,
		userID, "test-user-"+userID.String()[:8],
	)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	wsID := uuid.New()
	_, err = pool.Exec(context.Background(),
		`INSERT INTO workspaces (id, user_id, name, state) VALUES ($1, $2, $3, 'running')`,
		wsID, userID, "test-ws-"+wsID.String()[:8],
	)
	if err != nil {
		t.Fatalf("failed to create test workspace: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM workspaces WHERE id = $1`, wsID)
		pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})
	return wsID
}

func TestCreateFile(t *testing.T) {
	pool := getTestPool(t)
	if pool == nil {
		return
	}

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	wsID := createTestWorkspace(t, pool)
	now := time.Now()
	f := &FileMetadata{
		ID:            uuid.New(),
		WorkspaceID:   wsID,
		Filename:      "test_" + uuid.New().String()[:8] + ".enc",
		Size:          2048,
		ContentType:   "application/octet-stream",
		EncryptedPath: wsID.String() + "/" + uuid.New().String(),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	err := repo.CreateFile(ctx, f)
	if err != nil {
		t.Fatalf("CreateFile failed: %v", err)
	}

	found, err := repo.GetFile(ctx, f.ID)
	if err != nil {
		t.Fatalf("GetFile failed: %v", err)
	}

	if found.Filename != f.Filename {
		t.Errorf("expected Filename %s, got %s", f.Filename, found.Filename)
	}
	if found.Size != f.Size {
		t.Errorf("expected Size %d, got %d", f.Size, found.Size)
	}
}

func TestGetFilesByWorkspace(t *testing.T) {
	pool := getTestPool(t)
	if pool == nil {
		return
	}

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	wsID := createTestWorkspace(t, pool)
	now := time.Now()

	for i := 0; i < 3; i++ {
		f := &FileMetadata{
			ID:            uuid.New(),
			WorkspaceID:   wsID,
			Filename:      "file_" + uuid.New().String()[:8] + ".enc",
			Size:          1024,
			ContentType:   "application/octet-stream",
			EncryptedPath: wsID.String() + "/" + uuid.New().String(),
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := repo.CreateFile(ctx, f); err != nil {
			t.Fatalf("CreateFile failed: %v", err)
		}
	}

	files, err := repo.GetFilesByWorkspace(ctx, wsID)
	if err != nil {
		t.Fatalf("GetFilesByWorkspace failed: %v", err)
	}

	if len(files) < 3 {
		t.Errorf("expected at least 3 files, got %d", len(files))
	}
}

func TestDeleteFile(t *testing.T) {
	pool := getTestPool(t)
	if pool == nil {
		return
	}

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	wsID := createTestWorkspace(t, pool)
	now := time.Now()
	f := &FileMetadata{
		ID:            uuid.New(),
		WorkspaceID:   wsID,
		Filename:      "delete_test_" + uuid.New().String()[:8] + ".enc",
		Size:          512,
		ContentType:   "application/octet-stream",
		EncryptedPath: wsID.String() + "/" + uuid.New().String(),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	err := repo.CreateFile(ctx, f)
	if err != nil {
		t.Fatalf("CreateFile failed: %v", err)
	}

	err = repo.DeleteFile(ctx, f.ID)
	if err != nil {
		t.Fatalf("DeleteFile failed: %v", err)
	}

	_, err = repo.GetFile(ctx, f.ID)
	if err == nil {
		t.Error("expected error when getting deleted file")
	}
}