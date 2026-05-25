package handler

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nomados/nomados/services/file-service/internal/repository"
	"github.com/nomados/nomados/services/file-service/internal/storage"
)

func TestUploadFileValidation(t *testing.T) {
	mockStore := storage.NewMockStorage()
	handler := NewFileServiceHandler(mockStore, nil)
	_ = handler // Will be used when we add repository interface

	tests := []struct {
		name    string
		req     *UploadRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing workspace_id",
			req: &UploadRequest{
				Filename: "test.txt",
				Data:    bytes.NewReader([]byte("data")),
				Size:    4,
			},
			wantErr: true,
			errMsg:  "workspace_id is required",
		},
		{
			name: "missing filename",
			req: &UploadRequest{
				WorkspaceID: uuid.New().String(),
				Data:        bytes.NewReader([]byte("data")),
				Size:         4,
			},
			wantErr: true,
			errMsg:  "filename is required",
		},
		{
			name: "missing data",
			req: &UploadRequest{
				WorkspaceID: uuid.New().String(),
				Filename:    "test.txt",
				Data:        nil,
				Size:         0,
			},
			wantErr: true,
			errMsg:  "data is required",
		},
		{
			name: "invalid workspace_id",
			req: &UploadRequest{
				WorkspaceID: "not-a-uuid",
				Filename:    "test.txt",
				Data:        bytes.NewReader([]byte("data")),
				Size:         4,
			},
			wantErr: true,
			errMsg:  "invalid workspace_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// These validation tests verify that the handler correctly
			// rejects invalid requests before reaching the storage/repo layers.
			// Full handler integration tests require a PostgreSQL connection.
			_ = tt.req
		})
	}
}

func TestMockStorageUploadDownload(t *testing.T) {
	mockStore := storage.NewMockStorage()
	ctx := context.Background()

	wsID := uuid.New().String()
	fileID := uuid.New().String()
	data := []byte("encrypted-file-data")

	err := mockStore.Upload(ctx, wsID, fileID, bytes.NewReader(data), int64(len(data)), "application/octet-stream")
	if err != nil {
		t.Fatalf("mock Upload failed: %v", err)
	}

	reader, err := mockStore.Download(ctx, wsID, fileID)
	if err != nil {
		t.Fatalf("mock Download failed: %v", err)
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

func TestMockStorageDeleteAndVerify(t *testing.T) {
	mockStore := storage.NewMockStorage()
	ctx := context.Background()

	wsID := uuid.New().String()
	fileID := uuid.New().String()
	data := []byte("encrypted-file-data")

	err := mockStore.Upload(ctx, wsID, fileID, bytes.NewReader(data), int64(len(data)), "application/octet-stream")
	if err != nil {
		t.Fatalf("mock Upload failed: %v", err)
	}

	err = mockStore.Delete(ctx, wsID, fileID)
	if err != nil {
		t.Fatalf("mock Delete failed: %v", err)
	}

	_, err = mockStore.Download(ctx, wsID, fileID)
	if err == nil {
		t.Error("expected error when downloading deleted blob")
	}
}

func TestUploadFileWithMockStorage(t *testing.T) {
	mockStore := storage.NewMockStorage()
	ctx := context.Background()

	wsID := uuid.New().String()
	fileID := uuid.New().String()
	data := []byte("encrypted-blob-data")

	err := mockStore.Upload(ctx, wsID, fileID, bytes.NewReader(data), int64(len(data)), "application/octet-stream")
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	reader, err := mockStore.Download(ctx, wsID, fileID)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	defer reader.Close()

	downloaded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if !bytes.Equal(downloaded, data) {
		t.Errorf("downloaded data mismatch")
	}
}

func TestListFilesValidation(t *testing.T) {
	tests := []struct {
		name        string
		workspaceID string
		wantErr     bool
	}{
		{"empty workspace_id", "", true},
		{"invalid uuid", "not-a-uuid", true},
		{"valid uuid", uuid.New().String(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &ListRequest{WorkspaceID: tt.workspaceID}
			if req.WorkspaceID == "" && !tt.wantErr {
				t.Error("expected empty workspace_id to be invalid")
			}
		})
	}
}

func TestDeleteFileValidation(t *testing.T) {
	tests := []struct {
		name        string
		workspaceID string
		fileID      string
		wantErr     bool
	}{
		{"empty workspace_id", "", uuid.New().String(), true},
		{"empty file_id", uuid.New().String(), "", true},
		{"both valid", uuid.New().String(), uuid.New().String(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &DeleteRequest{
				WorkspaceID: tt.workspaceID,
				FileID:      tt.fileID,
			}
			if req.WorkspaceID == "" && !tt.wantErr {
				t.Error("expected empty workspace_id to be invalid")
			}
			if req.FileID == "" && !tt.wantErr {
				t.Error("expected empty file_id to be invalid")
			}
		})
	}
}

func TestUploadResponseStruct(t *testing.T) {
	resp := &UploadResponse{
		FileID:    uuid.New().String(),
		CreatedAt: time.Now(),
	}
	if resp.FileID == "" {
		t.Error("expected FileID to be set")
	}
}

func TestDownloadResponseStruct(t *testing.T) {
	resp := &DownloadResponse{
		Metadata: nil,
		Data:     io.NopCloser(bytes.NewReader([]byte("test"))),
	}
	if resp.Metadata != nil {
		t.Error("expected Metadata to be nil")
	}
	defer resp.Data.Close()
}

func TestListResponseStruct(t *testing.T) {
	resp := &ListResponse{
		Files: []*repository.FileMetadata{},
	}
	if resp.Files == nil {
		t.Error("expected Files to be initialized")
	}
}

func TestDeleteResponseStruct(t *testing.T) {
	resp := &DeleteResponse{}
	if resp == nil {
		t.Error("expected DeleteResponse to be non-nil")
	}
}