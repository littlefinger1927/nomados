package turn

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config.ServerURL != "turn:localhost:3478" {
		t.Errorf("expected default ServerURL turn:localhost:3478, got %s", config.ServerURL)
	}
	if config.Username != "nomados" {
		t.Errorf("expected default Username nomados, got %s", config.Username)
	}
	if config.Password != "nomados_dev_secret" {
		t.Errorf("expected default Password nomados_dev_secret, got %s", config.Password)
	}
	if config.Realm != "nomados.dev" {
		t.Errorf("expected default Realm nomados.dev, got %s", config.Realm)
	}
}

func TestGenerateTURNCredentials(t *testing.T) {
	config := DefaultConfig()
	relay := NewTURNRelay(config)

	username, password, err := relay.GenerateTURNCredentials("user1", 1*time.Hour)
	if err != nil {
		t.Fatalf("GenerateTURNCredentials returned error: %v", err)
	}

	// Username should be in format "timestamp:originalUsername"
	parts := strings.SplitN(username, ":", 2)
	if len(parts) != 2 {
		t.Fatalf("expected username format 'timestamp:userid', got %s", username)
	}
	if parts[1] != "user1" {
		t.Errorf("expected userid part 'user1', got %s", parts[1])
	}

	// Password should be a valid base64-encoded HMAC-SHA256
	decoded, err := base64.StdEncoding.DecodeString(password)
	if err != nil {
		t.Fatalf("password is not valid base64: %v", err)
	}
	if len(decoded) != sha256.Size {
		t.Errorf("expected HMAC-SHA256 output of %d bytes, got %d", sha256.Size, len(decoded))
	}

	// Verify the HMAC independently
	mac := hmac.New(sha256.New, []byte(config.Password))
	mac.Write([]byte(username))
	expectedMAC := mac.Sum(nil)
	if !hmac.Equal(decoded, expectedMAC) {
		t.Error("generated password does not match expected HMAC")
	}
}

func TestGenerateTURNCredentialsEmptyUsername(t *testing.T) {
	config := DefaultConfig()
	relay := NewTURNRelay(config)

	_, _, err := relay.GenerateTURNCredentials("", 1*time.Hour)
	if err == nil {
		t.Error("expected error for empty username, got nil")
	}
}

func TestGenerateTURNCredentialsZeroTTL(t *testing.T) {
	config := DefaultConfig()
	relay := NewTURNRelay(config)

	_, _, err := relay.GenerateTURNCredentials("user1", 0)
	if err == nil {
		t.Error("expected error for zero TTL, got nil")
	}
}

func TestGenerateTURNCredentialsNegativeTTL(t *testing.T) {
	config := DefaultConfig()
	relay := NewTURNRelay(config)

	_, _, err := relay.GenerateTURNCredentials("user1", -1*time.Hour)
	if err == nil {
		t.Error("expected error for negative TTL, got nil")
	}
}

func TestICEServers(t *testing.T) {
	config := DefaultConfig()
	relay := NewTURNRelay(config)

	servers := relay.ICEServers()
	if len(servers) != 1 {
		t.Fatalf("expected 1 ICE server (TURN-only), got %d", len(servers))
	}

	server := servers[0]
	if len(server.URLs) != 1 {
		t.Errorf("expected 1 URL, got %d", len(server.URLs))
	}
	if server.URLs[0] != config.ServerURL {
		t.Errorf("expected URL %s, got %s", config.ServerURL, server.URLs[0])
	}
	if server.Username != config.Username {
		t.Errorf("expected username %s, got %s", config.Username, server.Username)
	}
	if server.Credential != config.Password {
		t.Errorf("expected credential %s, got %s", config.Password, server.Credential)
	}
}

func TestConfig(t *testing.T) {
	config := DefaultConfig()
	relay := NewTURNRelay(config)

	returned := relay.Config()
	if returned != config {
		t.Error("Config() should return the same config used in construction")
	}
}

func TestGenerateTURNCredentialsDifferentTTLLs(t *testing.T) {
	config := DefaultConfig()
	relay := NewTURNRelay(config)

	user1, _, _ := relay.GenerateTURNCredentials("user1", 1*time.Hour)
	user2, _, _ := relay.GenerateTURNCredentials("user2", 24*time.Hour)

	// Usernames should be different
	if user1 == user2 {
		t.Error("different users should produce different usernames")
	}
}

func TestCustomConfig(t *testing.T) {
	config := TURNConfig{
		ServerURL: "turn:custom.example.com:3478",
		Username:  "custom_user",
		Password:  "custom_secret",
		Realm:     "custom.realm",
	}
	relay := NewTURNRelay(config)

	servers := relay.ICEServers()
	if servers[0].URLs[0] != "turn:custom.example.com:3478" {
		t.Errorf("expected custom TURN URL, got %s", servers[0].URLs[0])
	}
}