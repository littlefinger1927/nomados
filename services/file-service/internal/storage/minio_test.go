package storage

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Endpoint != "localhost:9000" {
		t.Errorf("expected endpoint localhost:9000, got %s", cfg.Endpoint)
	}
	if cfg.AccessKey != "nomados" {
		t.Errorf("expected access key nomados, got %s", cfg.AccessKey)
	}
	if cfg.SecretKey != "nomados_dev_key" {
		t.Errorf("expected secret key nomados_dev_key, got %s", cfg.SecretKey)
	}
	if cfg.Bucket != "nomados-files" {
		t.Errorf("expected bucket nomados-files, got %s", cfg.Bucket)
	}
	if cfg.UseSSL {
		t.Error("expected UseSSL to be false for dev config")
	}
}

func TestObjectPath(t *testing.T) {
	tests := []struct {
		workspaceID string
		fileID      string
		expected    string
	}{
		{"ws-123", "file-456", "ws-123/file-456"},
		{"a", "b", "a/b"},
		{"workspace-with-dash", "file-with-dash", "workspace-with-dash/file-with-dash"},
	}

	for _, tt := range tests {
		result := objectPath(tt.workspaceID, tt.fileID)
		if result != tt.expected {
			t.Errorf("objectPath(%s, %s) = %s, want %s", tt.workspaceID, tt.fileID, result, tt.expected)
		}
	}
}

func TestMockStorageUploadDownload(t *testing.T) {
	mock := NewMockStorage()
	ctx := context.Background()

	data := []byte("encrypted-file-content-here")
	err := mock.Upload(ctx, "ws-1", "file-1", bytes.NewReader(data), int64(len(data)), "application/octet-stream")
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	reader, err := mock.Download(ctx, "ws-1", "file-1")
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	defer reader.Close()

	downloaded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if !bytes.Equal(downloaded, data) {
		t.Errorf("downloaded data does not match uploaded data")
	}
}

func TestMockStorageDelete(t *testing.T) {
	mock := NewMockStorage()
	ctx := context.Background()

	data := []byte("encrypted-file-content")
	err := mock.Upload(ctx, "ws-1", "file-1", bytes.NewReader(data), int64(len(data)), "application/octet-stream")
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	err = mock.Delete(ctx, "ws-1", "file-1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = mock.Download(ctx, "ws-1", "file-1")
	if err == nil {
		t.Error("expected error when downloading deleted blob")
	}
}

func TestMockStorageDownloadNotFound(t *testing.T) {
	mock := NewMockStorage()
	ctx := context.Background()

	_, err := mock.Download(ctx, "ws-nonexistent", "file-nonexistent")
	if err == nil {
		t.Error("expected error when downloading non-existent blob")
	}
}

func TestStorageInterfaceCompliance(t *testing.T) {
	// Verify that MinIOStorage implements the Storage interface at compile time.
	var _ Storage = (*MinIOStorage)(nil)
	// Verify that MockStorage implements the Storage interface at compile time.
	var _ Storage = (*MockStorage)(nil)
}