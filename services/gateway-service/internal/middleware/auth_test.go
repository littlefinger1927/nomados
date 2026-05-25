package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authsdk "github.com/nomados/nomados/packages/auth-sdk"
)

func TestAuthMiddleware_ValidToken(t *testing.T) {
	validator := authsdk.NewTokenValidator("test-secret")
	token, _ := validator.GenerateAccessToken("user-1", "session-1", "device-1", 5*time.Minute)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if GetUserID(r.Context()) != "user-1" {
			t.Error("user ID not in context")
		}
		if GetSessionID(r.Context()) != "session-1" {
			t.Error("session ID not in context")
		}
		if GetDeviceID(r.Context()) != "device-1" {
			t.Error("device ID not in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := AuthMiddleware(validator, next)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("next handler was not called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	validator := authsdk.NewTokenValidator("test-secret")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	handler := AuthMiddleware(validator, next)
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	validator := authsdk.NewTokenValidator("test-secret")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	handler := AuthMiddleware(validator, next)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_BadFormat(t *testing.T) {
	validator := authsdk.NewTokenValidator("test-secret")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	handler := AuthMiddleware(validator, next)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic sometoken")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_WrongSecret(t *testing.T) {
	validator := authsdk.NewTokenValidator("test-secret")
	otherValidator := authsdk.NewTokenValidator("other-secret")
	token, _ := otherValidator.GenerateAccessToken("user-1", "session-1", "device-1", 5*time.Minute)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	handler := AuthMiddleware(validator, next)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}