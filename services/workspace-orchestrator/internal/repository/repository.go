package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nomados/nomados/packages/logging"
)

// WorkspaceRow represents a workspace row in the database.
type WorkspaceRow struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Name      string
	State     string
	NoVNCPort int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PostgresRepository implements workspace persistence with PostgreSQL.
type PostgresRepository struct {
	pool   *pgxpool.Pool
	logger *logging.Logger
}

// NewPostgresRepository creates a new PostgresRepository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		pool:   pool,
		logger: logging.NewLogger("workspace-repository", nil),
	}
}

// CreateWorkspace inserts a new workspace row and returns it.
func (r *PostgresRepository) CreateWorkspace(ctx context.Context, userID uuid.UUID, name, state string, novncPort int) (*WorkspaceRow, error) {
	var ws WorkspaceRow
	err := r.pool.QueryRow(ctx,
		`INSERT INTO workspaces (user_id, name, state, novnc_port) VALUES ($1, $2, $3, $4) RETURNING id, user_id, name, state, novnc_port, created_at, updated_at`,
		userID, name, state, novncPort,
	).Scan(&ws.ID, &ws.UserID, &ws.Name, &ws.State, &ws.NoVNCPort, &ws.CreatedAt, &ws.UpdatedAt)
	if err != nil {
		r.logger.Error("failed to create workspace", "error", err)
		return nil, err
	}
	return &ws, nil
}

// GetWorkspace returns a workspace by ID.
func (r *PostgresRepository) GetWorkspace(ctx context.Context, id uuid.UUID) (*WorkspaceRow, error) {
	var ws WorkspaceRow
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, name, state, novnc_port, created_at, updated_at FROM workspaces WHERE id = $1`,
		id,
	).Scan(&ws.ID, &ws.UserID, &ws.Name, &ws.State, &ws.NoVNCPort, &ws.CreatedAt, &ws.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &ws, nil
}

// ListWorkspacesByUser returns all workspaces for a given user.
func (r *PostgresRepository) ListWorkspacesByUser(ctx context.Context, userID uuid.UUID) ([]*WorkspaceRow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, name, state, novnc_port, created_at, updated_at FROM workspaces WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workspaces []*WorkspaceRow
	for rows.Next() {
		var ws WorkspaceRow
		if err := rows.Scan(&ws.ID, &ws.UserID, &ws.Name, &ws.State, &ws.NoVNCPort, &ws.CreatedAt, &ws.UpdatedAt); err != nil {
			return nil, err
		}
		workspaces = append(workspaces, &ws)
	}
	return workspaces, nil
}

// UpdateWorkspaceState updates the state and updated_at timestamp.
func (r *PostgresRepository) UpdateWorkspaceState(ctx context.Context, id uuid.UUID, state string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE workspaces SET state = $1, updated_at = NOW() WHERE id = $2`,
		state, id,
	)
	return err
}

// UpdateWorkspaceNoVNCPort updates the novnc_port for a workspace.
func (r *PostgresRepository) UpdateWorkspaceNoVNCPort(ctx context.Context, id uuid.UUID, port int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE workspaces SET novnc_port = $1, updated_at = NOW() WHERE id = $2`,
		port, id,
	)
	return err
}

// DeleteWorkspace removes a workspace row.
func (r *PostgresRepository) DeleteWorkspace(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM workspaces WHERE id = $1`,
		id,
	)
	return err
}