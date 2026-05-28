package handler

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nomados/nomados/packages/logging"
)

// DBOwnerChecker checks workspace ownership by querying the database directly.
// Both the file service and workspace orchestrator share the same PostgreSQL database.
type DBOwnerChecker struct {
	pool   *pgxpool.Pool
	logger *logging.Logger
}

// NewDBOwnerChecker creates a new DBOwnerChecker.
func NewDBOwnerChecker(pool *pgxpool.Pool) *DBOwnerChecker {
	return &DBOwnerChecker{
		pool:   pool,
		logger: logging.NewLogger("ownership-checker", nil),
	}
}

// IsWorkspaceOwner checks if the given user owns the specified workspace.
func (c *DBOwnerChecker) IsWorkspaceOwner(ctx context.Context, workspaceID, userID string) (bool, error) {
	var ownerID string
	err := c.pool.QueryRow(ctx,
		`SELECT user_id FROM workspaces WHERE id = $1`,
		workspaceID,
	).Scan(&ownerID)
	if err != nil {
		c.logger.Error("failed to query workspace ownership", "workspace_id", workspaceID, "error", err)
		return false, err
	}
	return ownerID == userID, nil
}