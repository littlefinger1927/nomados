package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nomados/nomados/packages/logging"
)

// FileMetadata represents metadata for an encrypted file stored in the system.
type FileMetadata struct {
	ID           uuid.UUID `json:"id"`
	WorkspaceID  uuid.UUID `json:"workspace_id"`
	Filename     string    `json:"filename"`
	Size         int64     `json:"size"`
	ContentType  string    `json:"content_type"`
	EncryptedPath string   `json:"encrypted_path"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PostgresRepository implements file metadata storage against PostgreSQL.
type PostgresRepository struct {
	pool   *pgxpool.Pool
	logger *logging.Logger
}

// NewPostgresRepository creates a new PostgresRepository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		pool:   pool,
		logger: logging.NewLogger("file-repository", nil),
	}
}

// CreateFile inserts a new file metadata record into the database.
func (r *PostgresRepository) CreateFile(ctx context.Context, metadata *FileMetadata) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO files (id, workspace_id, filename, size, content_type, encrypted_path, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		metadata.ID, metadata.WorkspaceID, metadata.Filename, metadata.Size,
		metadata.ContentType, metadata.EncryptedPath, metadata.CreatedAt, metadata.UpdatedAt,
	)
	if err != nil {
		r.logger.Error("failed to create file metadata", "file_id", metadata.ID, "error", err)
		return err
	}

	r.logger.Info("file metadata created", "file_id", metadata.ID, "workspace_id", metadata.WorkspaceID)
	return nil
}

// GetFile retrieves file metadata by ID.
func (r *PostgresRepository) GetFile(ctx context.Context, id uuid.UUID) (*FileMetadata, error) {
	var f FileMetadata
	err := r.pool.QueryRow(ctx,
		`SELECT id, workspace_id, filename, size, content_type, encrypted_path, created_at, updated_at
		 FROM files WHERE id = $1`,
		id,
	).Scan(&f.ID, &f.WorkspaceID, &f.Filename, &f.Size, &f.ContentType, &f.EncryptedPath, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		r.logger.Error("failed to get file metadata", "file_id", id, "error", err)
		return nil, err
	}

	return &f, nil
}

// GetFilesByWorkspace retrieves all file metadata for a given workspace.
func (r *PostgresRepository) GetFilesByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]*FileMetadata, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, workspace_id, filename, size, content_type, encrypted_path, created_at, updated_at
		 FROM files WHERE workspace_id = $1 ORDER BY created_at DESC`,
		workspaceID,
	)
	if err != nil {
		r.logger.Error("failed to list files by workspace", "workspace_id", workspaceID, "error", err)
		return nil, err
	}
	defer rows.Close()

	var files []*FileMetadata
	for rows.Next() {
		var f FileMetadata
		if err := rows.Scan(&f.ID, &f.WorkspaceID, &f.Filename, &f.Size, &f.ContentType, &f.EncryptedPath, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		files = append(files, &f)
	}

	return files, rows.Err()
}

// DeleteFile removes file metadata from the database.
func (r *PostgresRepository) DeleteFile(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM files WHERE id = $1`,
		id,
	)
	if err != nil {
		r.logger.Error("failed to delete file metadata", "file_id", id, "error", err)
		return err
	}

	r.logger.Info("file metadata deleted", "file_id", id)
	return nil
}