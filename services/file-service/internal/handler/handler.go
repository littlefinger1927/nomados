package handler

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/nomados/nomados/packages/logging"
	"github.com/nomados/nomados/services/file-service/internal/repository"
	"github.com/nomados/nomados/services/file-service/internal/storage"
)

// Request and response types for Phase 1 (struct-based, until proto stubs are generated).

// UploadRequest represents a file upload request.
type UploadRequest struct {
	WorkspaceID string
	Filename    string
	ContentType string
	Data        io.Reader
	Size        int64
}

// UploadResponse represents a file upload response.
type UploadResponse struct {
	FileID    string
	CreatedAt time.Time
}

// DownloadRequest represents a file download request.
type DownloadRequest struct {
	WorkspaceID string
	FileID      string
}

// DownloadResponse represents a file download response.
type DownloadResponse struct {
	Metadata   *repository.FileMetadata
	Data       io.ReadCloser
}

// ListRequest represents a request to list files in a workspace.
type ListRequest struct {
	WorkspaceID string
}

// ListResponse represents a list of file metadata.
type ListResponse struct {
	Files []*repository.FileMetadata
}

// DeleteRequest represents a file delete request.
type DeleteRequest struct {
	WorkspaceID string
	FileID      string
}

// DeleteResponse represents a file delete response.
type DeleteResponse struct{}

// FileServiceHandler handles file storage operations.
type FileServiceHandler struct {
	store    storage.Storage
	repo     *repository.PostgresRepository
	logger   *logging.Logger
}

// NewFileServiceHandler creates a new FileServiceHandler.
func NewFileServiceHandler(store storage.Storage, repo *repository.PostgresRepository) *FileServiceHandler {
	return &FileServiceHandler{
		store:  store,
		repo:   repo,
		logger: logging.NewLogger("file-handler", nil),
	}
}

// UploadFile validates input, stores the encrypted blob in MinIO, and saves metadata to PostgreSQL.
func (h *FileServiceHandler) UploadFile(ctx context.Context, req *UploadRequest) (*UploadResponse, error) {
	if req.WorkspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required")
	}
	if req.Filename == "" {
		return nil, fmt.Errorf("filename is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("data is required")
	}

	wsID, err := uuid.Parse(req.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace_id: %w", err)
	}

	fileID := uuid.New()
	now := time.Now()
	encryptedPath := fmt.Sprintf("%s/%s", wsID, fileID)

	// Store the encrypted blob in MinIO.
	if err := h.store.Upload(ctx, wsID.String(), fileID.String(), req.Data, req.Size, req.ContentType); err != nil {
		h.logger.Error("failed to store blob", "file_id", fileID, "error", err)
		return nil, fmt.Errorf("failed to store blob: %w", err)
	}

	// Store file metadata in PostgreSQL.
	contentType := req.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	metadata := &repository.FileMetadata{
		ID:            fileID,
		WorkspaceID:   wsID,
		Filename:      req.Filename,
		Size:          req.Size,
		ContentType:   contentType,
		EncryptedPath: encryptedPath,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := h.repo.CreateFile(ctx, metadata); err != nil {
		// Attempt to clean up the blob from MinIO since metadata save failed.
		if delErr := h.store.Delete(ctx, wsID.String(), fileID.String()); delErr != nil {
			h.logger.Error("failed to clean up blob after metadata save failure", "file_id", fileID, "error", delErr)
		}
		h.logger.Error("failed to save file metadata", "file_id", fileID, "error", err)
		return nil, fmt.Errorf("failed to save file metadata: %w", err)
	}

	h.logger.Info("file uploaded", "file_id", fileID, "workspace_id", wsID, "filename", req.Filename)

	return &UploadResponse{
		FileID:    fileID.String(),
		CreatedAt: now,
	}, nil
}

// DownloadFile fetches file metadata from PostgreSQL and streams the encrypted blob from MinIO.
func (h *FileServiceHandler) DownloadFile(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
	if req.WorkspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required")
	}
	if req.FileID == "" {
		return nil, fmt.Errorf("file_id is required")
	}

	fileID, err := uuid.Parse(req.FileID)
	if err != nil {
		return nil, fmt.Errorf("invalid file_id: %w", err)
	}

	// Fetch metadata from PostgreSQL.
	metadata, err := h.repo.GetFile(ctx, fileID)
	if err != nil {
		h.logger.Error("failed to get file metadata", "file_id", fileID, "error", err)
		return nil, fmt.Errorf("file not found: %w", err)
	}

	// Verify workspace ownership.
	wsID, err := uuid.Parse(req.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace_id: %w", err)
	}
	if metadata.WorkspaceID != wsID {
		return nil, fmt.Errorf("file does not belong to workspace")
	}

	// Download the encrypted blob from MinIO.
	data, err := h.store.Download(ctx, wsID.String(), fileID.String())
	if err != nil {
		h.logger.Error("failed to download blob", "file_id", fileID, "error", err)
		return nil, fmt.Errorf("failed to download blob: %w", err)
	}

	h.logger.Info("file downloaded", "file_id", fileID, "workspace_id", wsID)

	return &DownloadResponse{
		Metadata: metadata,
		Data:     data,
	}, nil
}

// ListFiles returns all file metadata for a given workspace.
func (h *FileServiceHandler) ListFiles(ctx context.Context, req *ListRequest) (*ListResponse, error) {
	if req.WorkspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required")
	}

	wsID, err := uuid.Parse(req.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace_id: %w", err)
	}

	files, err := h.repo.GetFilesByWorkspace(ctx, wsID)
	if err != nil {
		h.logger.Error("failed to list files", "workspace_id", wsID, "error", err)
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	h.logger.Info("files listed", "workspace_id", wsID, "count", len(files))

	return &ListResponse{Files: files}, nil
}

// DeleteFile removes the encrypted blob from MinIO and deletes metadata from PostgreSQL.
func (h *FileServiceHandler) DeleteFile(ctx context.Context, req *DeleteRequest) (*DeleteResponse, error) {
	if req.WorkspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required")
	}
	if req.FileID == "" {
		return nil, fmt.Errorf("file_id is required")
	}

	fileID, err := uuid.Parse(req.FileID)
	if err != nil {
		return nil, fmt.Errorf("invalid file_id: %w", err)
	}

	wsID, err := uuid.Parse(req.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace_id: %w", err)
	}

	// Verify file exists and belongs to workspace.
	metadata, err := h.repo.GetFile(ctx, fileID)
	if err != nil {
		h.logger.Error("failed to get file metadata for deletion", "file_id", fileID, "error", err)
		return nil, fmt.Errorf("file not found: %w", err)
	}

	if metadata.WorkspaceID != wsID {
		return nil, fmt.Errorf("file does not belong to workspace")
	}

	// Delete blob from MinIO.
	if err := h.store.Delete(ctx, wsID.String(), fileID.String()); err != nil {
		h.logger.Error("failed to delete blob", "file_id", fileID, "error", err)
		return nil, fmt.Errorf("failed to delete blob: %w", err)
	}

	// Delete metadata from PostgreSQL.
	if err := h.repo.DeleteFile(ctx, fileID); err != nil {
		h.logger.Error("failed to delete file metadata", "file_id", fileID, "error", err)
		return nil, fmt.Errorf("failed to delete file metadata: %w", err)
	}

	h.logger.Info("file deleted", "file_id", fileID, "workspace_id", wsID)

	return &DeleteResponse{}, nil
}