package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Session represents a session record from the database.
type Session struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	DeviceID  uuid.UUID `json:"device_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	RiskScore int       `json:"risk_score"`
	IPHash    string    `json:"ip_hash"`
	Revoked   bool      `json:"revoked"`
}

// PostgresRepository implements data access against PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgresRepository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// CreateSession inserts a new session into the database.
func (r *PostgresRepository) CreateSession(ctx context.Context, userID, deviceID, ipHash string, riskScore int, expiresAt time.Time) (*Session, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	var s Session
	err := r.pool.QueryRow(ctx,
		`INSERT INTO sessions (user_id, device_id, ip_hash, risk_score, expires_at) VALUES ($1, $2, $3, $4, $5) RETURNING id, user_id, device_id, created_at, expires_at, risk_score, ip_hash, revoked`,
		userID, deviceID, ipHash, riskScore, expiresAt,
	).Scan(&s.ID, &s.UserID, &s.DeviceID, &s.CreatedAt, &s.ExpiresAt, &s.RiskScore, &s.IPHash, &s.Revoked)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetSessionByID retrieves a session by its ID.
func (r *PostgresRepository) GetSessionByID(ctx context.Context, id string) (*Session, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	var s Session
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, device_id, created_at, expires_at, risk_score, ip_hash, revoked FROM sessions WHERE id = $1`,
		id,
	).Scan(&s.ID, &s.UserID, &s.DeviceID, &s.CreatedAt, &s.ExpiresAt, &s.RiskScore, &s.IPHash, &s.Revoked)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetSessionsByUserID retrieves all sessions for a user.
func (r *PostgresRepository) GetSessionsByUserID(ctx context.Context, userID string) ([]*Session, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, device_id, created_at, expires_at, risk_score, ip_hash, revoked FROM sessions WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.ID, &s.UserID, &s.DeviceID, &s.CreatedAt, &s.ExpiresAt, &s.RiskScore, &s.IPHash, &s.Revoked); err != nil {
			return nil, err
		}
		sessions = append(sessions, &s)
	}
	return sessions, rows.Err()
}

// ValidateSession returns a session only if it is not revoked and not expired.
func (r *PostgresRepository) ValidateSession(ctx context.Context, id string) (*Session, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	var s Session
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, device_id, created_at, expires_at, risk_score, ip_hash, revoked FROM sessions WHERE id = $1 AND revoked = false AND expires_at > NOW()`,
		id,
	).Scan(&s.ID, &s.UserID, &s.DeviceID, &s.CreatedAt, &s.ExpiresAt, &s.RiskScore, &s.IPHash, &s.Revoked)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// RevokeSession marks a session as revoked.
func (r *PostgresRepository) RevokeSession(ctx context.Context, id string) error {
	if r.pool == nil {
		return fmt.Errorf("database connection not available")
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE sessions SET revoked = true WHERE id = $1`,
		id,
	)
	return err
}

// RevokeSessionsByUserID revokes all sessions for a given user.
func (r *PostgresRepository) RevokeSessionsByUserID(ctx context.Context, userID string) error {
	if r.pool == nil {
		return fmt.Errorf("database connection not available")
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE sessions SET revoked = true WHERE user_id = $1`,
		userID,
	)
	return err
}

// UpdateRiskScore updates the risk score of a session.
func (r *PostgresRepository) UpdateRiskScore(ctx context.Context, id string, riskScore int) error {
	if r.pool == nil {
		return fmt.Errorf("database connection not available")
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE sessions SET risk_score = $1 WHERE id = $2`,
		riskScore, id,
	)
	return err
}