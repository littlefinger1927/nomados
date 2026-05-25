package turn

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"time"
)

// TURNConfig holds TURN server configuration.
type TURNConfig struct {
	ServerURL string
	Username  string
	Password  string
	Realm     string
}

// DefaultConfig returns the default TURN configuration using environment
// variables or development defaults.
func DefaultConfig() TURNConfig {
	return TURNConfig{
		ServerURL: envOrDefault("TURN_SERVER_URL", "turn:localhost:3478"),
		Username:  envOrDefault("TURN_USERNAME", "nomados"),
		Password:  envOrDefault("TURN_PASSWORD", "nomados_dev_secret"),
		Realm:     envOrDefault("TURN_REALM", "nomados.dev"),
	}
}

// TURNRelay provides TURN credential generation and configuration.
type TURNRelay struct {
	config TURNConfig
}

// NewTURNRelay creates a new TURNRelay with the given configuration.
func NewTURNRelay(config TURNConfig) *TURNRelay {
	return &TURNRelay{config: config}
}

// Config returns the TURN configuration.
func (r *TURNRelay) Config() TURNConfig {
	return r.config
}

// GenerateTURNCredentials generates time-limited TURN credentials using
// HMAC-SHA256. This implements the TURN long-term credential mechanism
// where the password is computed as:
//
//	password = HMAC-SHA256(shared_secret, username)
//
// The username format includes a timestamp: "timestamp:userid"
// The resulting credential is valid for the specified TTL.
func (r *TURNRelay) GenerateTURNCredentials(username string, ttl time.Duration) (string, string, error) {
	if username == "" {
		return "", "", fmt.Errorf("username must not be empty")
	}
	if ttl <= 0 {
		return "", "", fmt.Errorf("ttl must be positive")
	}

	// TURN long-term credential: username is "expiryTimestamp:originalUsername"
	expiry := time.Now().UTC().Add(ttl).Unix()
	turnUsername := fmt.Sprintf("%d:%s", expiry, username)

	// HMAC-SHA256 with the shared secret to generate the password
	mac := hmac.New(sha256.New, []byte(r.config.Password))
	mac.Write([]byte(turnUsername))
	turnPassword := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return turnUsername, turnPassword, nil
}

// ICEServers returns the ICE server configuration with TURN-only servers
// (no STUN, per the TURN-only relay requirement).
func (r *TURNRelay) ICEServers() []ICEServer {
	return []ICEServer{
		{
			URLs:       []string{r.config.ServerURL},
			Username:   r.config.Username,
			Credential: r.config.Password,
		},
	}
}

// ICEServer represents a single ICE server configuration.
type ICEServer struct {
	URLs       []string
	Username   string
	Credential string
}

func envOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}