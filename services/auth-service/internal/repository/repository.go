package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// User represents a user record from the database.
type User struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// Device represents a device record from the database.
type Device struct {
	ID                   uuid.UUID  `json:"id"`
	UserID               uuid.UUID  `json:"user_id"`
	PublicKey            []byte     `json:"public_key"`
	Attestation          string     `json:"attestation"`
	Trusted              bool       `json:"trusted"`
	LastSeen             *time.Time `json:"last_seen"`
	CreatedAt            time.Time  `json:"created_at"`
	CredentialID         []byte     `json:"credential_id"`
	CredentialPublicKey  []byte     `json:"credential_public_key"`
	SignCount            int64      `json:"sign_count"`
	Name                 string     `json:"name"`
	AAGUID               []byte     `json:"aaguid"`
}

// PostgresRepository implements data access against PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgresRepository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// CreateUser inserts a new user into the database.
func (r *PostgresRepository) CreateUser(ctx context.Context, username string) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (username, status) VALUES ($1, 'active') RETURNING id, username, status, created_at`,
		username,
	).Scan(&u.ID, &u.Username, &u.Status, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByID retrieves a user by their ID.
func (r *PostgresRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx,
		`SELECT id, username, status, created_at FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Username, &u.Status, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByUsername retrieves a user by their username.
func (r *PostgresRepository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx,
		`SELECT id, username, status, created_at FROM users WHERE username = $1`,
		username,
	).Scan(&u.ID, &u.Username, &u.Status, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateDevice inserts a new device into the database.
func (r *PostgresRepository) CreateDevice(ctx context.Context, userID uuid.UUID, publicKey []byte, attestation string) (*Device, error) {
	var d Device
	err := r.pool.QueryRow(ctx,
		`INSERT INTO devices (user_id, public_key, attestation, trusted) VALUES ($1, $2, $3, false) RETURNING id, user_id, public_key, attestation, trusted, last_seen, created_at, credential_id, credential_public_key, sign_count, name`,
		userID, publicKey, attestation,
	).Scan(&d.ID, &d.UserID, &d.PublicKey, &d.Attestation, &d.Trusted, &d.LastSeen, &d.CreatedAt, &d.CredentialID, &d.CredentialPublicKey, &d.SignCount, &d.Name)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// CreateDeviceWithName inserts a new device with a name into the database.
func (r *PostgresRepository) CreateDeviceWithName(ctx context.Context, userID uuid.UUID, publicKey []byte, attestation string, deviceName string) (*Device, error) {
	var d Device
	err := r.pool.QueryRow(ctx,
		`INSERT INTO devices (user_id, public_key, attestation, trusted, name) VALUES ($1, $2, $3, false, $4) RETURNING id, user_id, public_key, attestation, trusted, last_seen, created_at, credential_id, credential_public_key, sign_count, name`,
		userID, publicKey, attestation, deviceName,
	).Scan(&d.ID, &d.UserID, &d.PublicKey, &d.Attestation, &d.Trusted, &d.LastSeen, &d.CreatedAt, &d.CredentialID, &d.CredentialPublicKey, &d.SignCount, &d.Name)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// DeleteDevice deletes a device by its ID.
func (r *PostgresRepository) DeleteDevice(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM devices WHERE id = $1`, id)
	return err
}

// GetDeviceByID retrieves a device by its ID.
func (r *PostgresRepository) GetDeviceByID(ctx context.Context, id uuid.UUID) (*Device, error) {
	var d Device
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, public_key, attestation, trusted, last_seen, created_at, credential_id, credential_public_key, sign_count, name FROM devices WHERE id = $1`,
		id,
	).Scan(&d.ID, &d.UserID, &d.PublicKey, &d.Attestation, &d.Trusted, &d.LastSeen, &d.CreatedAt, &d.CredentialID, &d.CredentialPublicKey, &d.SignCount, &d.Name)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// GetDevicesByUserID retrieves all devices for a user.
func (r *PostgresRepository) GetDevicesByUserID(ctx context.Context, userID uuid.UUID) ([]*Device, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, public_key, attestation, trusted, last_seen, created_at, credential_id, credential_public_key, sign_count, name FROM devices WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []*Device
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.ID, &d.UserID, &d.PublicKey, &d.Attestation, &d.Trusted, &d.LastSeen, &d.CreatedAt, &d.CredentialID, &d.CredentialPublicKey, &d.SignCount, &d.Name); err != nil {
			return nil, err
		}
		devices = append(devices, &d)
	}
	return devices, rows.Err()
}

// UpdateDeviceLastSeen updates the last_seen timestamp of a device.
func (r *PostgresRepository) UpdateDeviceLastSeen(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE devices SET last_seen = NOW() WHERE id = $1`,
		id,
	)
	return err
}

// TrustDevice marks a device as trusted.
func (r *PostgresRepository) TrustDevice(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE devices SET trusted = true WHERE id = $1`,
		id,
	)
	return err
}

// GetDeviceByPublicKey retrieves a device by its public key.
func (r *PostgresRepository) GetDeviceByPublicKey(ctx context.Context, publicKey []byte) (*Device, error) {
	var d Device
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, public_key, attestation, trusted, last_seen, created_at, credential_id, credential_public_key, sign_count, name FROM devices WHERE public_key = $1`,
		publicKey,
	).Scan(&d.ID, &d.UserID, &d.PublicKey, &d.Attestation, &d.Trusted, &d.LastSeen, &d.CreatedAt, &d.CredentialID, &d.CredentialPublicKey, &d.SignCount, &d.Name)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// UpdateDeviceCredential updates the WebAuthn credential data for a device.
func (r *PostgresRepository) UpdateDeviceCredential(ctx context.Context, id uuid.UUID, credentialID []byte, credentialPublicKey []byte) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE devices SET credential_id = $1, credential_public_key = $2 WHERE id = $3`,
		credentialID, credentialPublicKey, id,
	)
	return err
}

// UpdateDeviceSignCount updates the sign count for a device (used for replay protection).
func (r *PostgresRepository) UpdateDeviceSignCount(ctx context.Context, id uuid.UUID, signCount int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE devices SET sign_count = $1 WHERE id = $2`,
		signCount, id,
	)
	return err
}

// CreateAuditLog inserts an audit log entry.
func (r *PostgresRepository) CreateAuditLog(ctx context.Context, actorID uuid.UUID, action, target string, details map[string]interface{}) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO audit_logs (actor_id, action, target, details) VALUES ($1, $2, $3, $4)`,
		actorID, action, target, detailsJSON,
	)
	return err
}