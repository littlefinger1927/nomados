package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/nomados/nomados/packages/logging"
)

// MinIOConfig holds configuration for connecting to a MinIO instance.
type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

// DefaultConfig returns a development-friendly MinIO configuration.
func DefaultConfig() MinIOConfig {
	return MinIOConfig{
		Endpoint:  "localhost:9000",
		AccessKey: "nomados",
		SecretKey: "nomados_dev_key",
		Bucket:    "nomados-files",
		UseSSL:    false,
	}
}

// Storage defines the interface for encrypted blob storage operations.
type Storage interface {
	Upload(ctx context.Context, workspaceID, fileID string, data io.Reader, size int64, contentType string) error
	Download(ctx context.Context, workspaceID, fileID string) (io.ReadCloser, error)
	Delete(ctx context.Context, workspaceID, fileID string) error
	EnsureBucket(ctx context.Context) error
}

// MinIOStorage implements the Storage interface using MinIO as the backend.
type MinIOStorage struct {
	client *minio.Client
	bucket string
	logger *logging.Logger
}

// NewMinIOStorage creates a new MinIOStorage from the given config.
func NewMinIOStorage(cfg MinIOConfig) (*MinIOStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	return &MinIOStorage{
		client: client,
		bucket: cfg.Bucket,
		logger: logging.NewLogger("file-storage", nil),
	}, nil
}

// objectPath constructs the MinIO object key from workspace and file IDs.
func objectPath(workspaceID, fileID string) string {
	return fmt.Sprintf("%s/%s", workspaceID, fileID)
}

// Upload stores an encrypted blob in MinIO under {workspaceID}/{fileID}.
func (s *MinIOStorage) Upload(ctx context.Context, workspaceID, fileID string, data io.Reader, size int64, contentType string) error {
	path := objectPath(workspaceID, fileID)

	opts := minio.PutObjectOptions{
		ContentType: contentType,
		UserMetadata: map[string]string{
			"X-Nomados-Workspace": workspaceID,
			"X-Nomados-File":      fileID,
		},
	}

	info, err := s.client.PutObject(ctx, s.bucket, path, data, size, opts)
	if err != nil {
		s.logger.Error("failed to upload blob", "workspace_id", workspaceID, "file_id", fileID, "error", err)
		return fmt.Errorf("failed to upload blob: %w", err)
	}

	s.logger.Info("blob uploaded", "workspace_id", workspaceID, "file_id", fileID, "etag", info.ETag)
	return nil
}

// Download retrieves an encrypted blob from MinIO.
func (s *MinIOStorage) Download(ctx context.Context, workspaceID, fileID string) (io.ReadCloser, error) {
	path := objectPath(workspaceID, fileID)

	obj, err := s.client.GetObject(ctx, s.bucket, path, minio.GetObjectOptions{})
	if err != nil {
		s.logger.Error("failed to download blob", "workspace_id", workspaceID, "file_id", fileID, "error", err)
		return nil, fmt.Errorf("failed to download blob: %w", err)
	}

	s.logger.Info("blob downloaded", "workspace_id", workspaceID, "file_id", fileID)
	return obj, nil
}

// Delete removes an encrypted blob from MinIO.
func (s *MinIOStorage) Delete(ctx context.Context, workspaceID, fileID string) error {
	path := objectPath(workspaceID, fileID)

	err := s.client.RemoveObject(ctx, s.bucket, path, minio.RemoveObjectOptions{})
	if err != nil {
		s.logger.Error("failed to delete blob", "workspace_id", workspaceID, "file_id", fileID, "error", err)
		return fmt.Errorf("failed to delete blob: %w", err)
	}

	s.logger.Info("blob deleted", "workspace_id", workspaceID, "file_id", fileID)
	return nil
}

// EnsureBucket creates the configured bucket in MinIO if it does not already exist.
func (s *MinIOStorage) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if exists {
		s.logger.Info("bucket already exists", "bucket", s.bucket)
		return nil
	}

	if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	s.logger.Info("bucket created", "bucket", s.bucket)
	return nil
}