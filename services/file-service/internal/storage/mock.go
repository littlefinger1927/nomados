package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
)

// MockStorage implements the Storage interface for testing without a real MinIO server.
type MockStorage struct {
	Blobs    map[string][]byte
	Contents map[string]string
}

// NewMockStorage creates a new MockStorage.
func NewMockStorage() *MockStorage {
	return &MockStorage{
		Blobs:    make(map[string][]byte),
		Contents: make(map[string]string),
	}
}

// Upload stores data in memory.
func (m *MockStorage) Upload(ctx context.Context, workspaceID, fileID string, data io.Reader, size int64, contentType string) error {
	key := objectPath(workspaceID, fileID)
	b, err := io.ReadAll(data)
	if err != nil {
		return err
	}
	m.Blobs[key] = b
	m.Contents[key] = contentType
	return nil
}

// Download retrieves data from memory.
func (m *MockStorage) Download(ctx context.Context, workspaceID, fileID string) (io.ReadCloser, error) {
	key := objectPath(workspaceID, fileID)
	b, ok := m.Blobs[key]
	if !ok {
		return nil, fmt.Errorf("blob not found: %s", key)
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

// Delete removes data from memory.
func (m *MockStorage) Delete(ctx context.Context, workspaceID, fileID string) error {
	key := objectPath(workspaceID, fileID)
	delete(m.Blobs, key)
	delete(m.Contents, key)
	return nil
}

// EnsureBucket is a no-op for the mock.
func (m *MockStorage) EnsureBucket(ctx context.Context) error {
	return nil
}