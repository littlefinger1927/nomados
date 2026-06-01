package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ListenAddr          string
	AuthAddr            string
	SessionAddr         string
	WorkspaceAddr       string
	BrowserAddr         string
	StreamingAddr       string
	FileAddr            string
	VaultAddr           string
	SigningSecret       string
	RateLimitRPS        float64
	RateLimitBurst      int
	CORSAllowedOrigins  []string
}

func Load() *Config {
	return &Config{
		ListenAddr:          getEnv("GATEWAY_ADDR", ":8080"),
		AuthAddr:            getEnv("AUTH_SERVICE_ADDR", "localhost:50051"),
		SessionAddr:         getEnv("SESSION_SERVICE_ADDR", "localhost:50052"),
		WorkspaceAddr:       getEnv("WORKSPACE_SERVICE_ADDR", "localhost:50053"),
		BrowserAddr:         getEnv("BROWSER_SERVICE_ADDR", "localhost:50054"),
		StreamingAddr:       getEnv("STREAMING_SERVICE_ADDR", "localhost:50055"),
		FileAddr:            getEnv("FILE_SERVICE_ADDR", "localhost:50056"),
		VaultAddr:           getEnv("VAULT_SERVICE_ADDR", "localhost:50057"),
		SigningSecret:       getEnv("SIGNING_SECRET", "nomados-dev-secret"),
		RateLimitRPS:        getEnvFloat("RATE_LIMIT_RPS", 100),
		RateLimitBurst:      getEnvInt("RATE_LIMIT_BURST", 200),
		CORSAllowedOrigins:  parseCSVEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func parseCSVEnv(key, fallback string) []string {
	v := getEnv(key, fallback)
	if v == "" {
		return nil
	}
	return strings.Split(v, ",")
}