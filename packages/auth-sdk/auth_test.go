package authsdk

import (
	"testing"
	"time"
)

func TestTokenValidation(t *testing.T) {
	validator := NewTokenValidator("test-secret")
	token, err := validator.GenerateAccessToken("user-1", "session-1", "device-1", 5*time.Minute)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	claims, err := validator.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}
	if claims.UserID != "user-1" {
		t.Errorf("expected user-1, got %s", claims.UserID)
	}
	if claims.SessionID != "session-1" {
		t.Errorf("expected session-1, got %s", claims.SessionID)
	}
}

func TestExpiredToken(t *testing.T) {
	validator := NewTokenValidator("test-secret")
	token, _ := validator.GenerateAccessToken("user-1", "session-1", "device-1", -1*time.Second)
	_, err := validator.ValidateAccessToken(token)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestInvalidSignature(t *testing.T) {
	validator1 := NewTokenValidator("secret-1")
	validator2 := NewTokenValidator("secret-2")
	token, _ := validator1.GenerateAccessToken("user-1", "session-1", "device-1", 5*time.Minute)
	_, err := validator2.ValidateAccessToken(token)
	if err == nil {
		t.Error("expected error for invalid signature")
	}
}