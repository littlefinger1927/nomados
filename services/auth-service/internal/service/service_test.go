package service

import (
	"context"
	"crypto/rand"
	"testing"
	"time"
)

func newTestStore() *MemoryChallengeStore {
	return NewMemoryChallengeStore()
}

func TestGenerateRegistrationChallenge(t *testing.T) {
	store := newTestStore()
	defer store.Close()

	challenge, err := generateRegistrationChallenge(context.Background(), store, "test-user-id")
	if err != nil {
		t.Fatalf("generateRegistrationChallenge failed: %v", err)
	}

	if len(challenge) != 32 {
		t.Errorf("expected 32-byte challenge, got %d bytes", len(challenge))
	}
}

func TestGenerateRegistrationChallengeUniqueness(t *testing.T) {
	store := newTestStore()
	defer store.Close()

	ch1, _ := generateRegistrationChallenge(context.Background(), store, "user-1")
	ch2, _ := generateRegistrationChallenge(context.Background(), store, "user-1")

	// Challenges should be different (extremely unlikely to collide)
	same := true
	for i := range ch1 {
		if ch1[i] != ch2[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("expected different challenges for consecutive calls")
	}
}

func TestVerifyRegistrationCredential(t *testing.T) {
	err := verifyRegistrationCredential([]byte("cred"), []byte("challenge"))
	if err != nil {
		t.Fatalf("verifyRegistrationCredential stub should always succeed, got: %v", err)
	}
}

func TestGenerateAssertionChallenge(t *testing.T) {
	store := newTestStore()
	defer store.Close()

	challenge, err := generateAssertionChallenge(context.Background(), store, "test-user-id")
	if err != nil {
		t.Fatalf("generateAssertionChallenge failed: %v", err)
	}

	if len(challenge) != 32 {
		t.Errorf("expected 32-byte challenge, got %d bytes", len(challenge))
	}
}

func TestVerifyAssertionCredential(t *testing.T) {
	err := verifyAssertionCredential([]byte("assertion"), []byte("challenge"))
	if err != nil {
		t.Fatalf("verifyAssertionCredential stub should always succeed, got: %v", err)
	}
}

func TestGenerateAssertionChallengeUniqueness(t *testing.T) {
	store := newTestStore()
	defer store.Close()

	ch1, _ := generateAssertionChallenge(context.Background(), store, "user-1")
	ch2, _ := generateAssertionChallenge(context.Background(), store, "user-1")

	same := true
	for i := range ch1 {
		if ch1[i] != ch2[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("expected different challenges for consecutive calls")
	}
}

func TestMemoryChallengeStore_PutGet(t *testing.T) {
	store := newTestStore()
	defer store.Close()
	ctx := context.Background()

	err := store.Put(ctx, "test-key", "test-value", 5*time.Minute)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	val, err := store.Get(ctx, "test-key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "test-value" {
		t.Errorf("expected 'test-value', got '%s'", val)
	}

	// Second Get should return empty (one-time use)
	val2, err := store.Get(ctx, "test-key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val2 != "" {
		t.Errorf("expected empty string on second Get (one-time use), got '%s'", val2)
	}
}

func TestMemoryChallengeStore_GetNonExistent(t *testing.T) {
	store := newTestStore()
	defer store.Close()
	ctx := context.Background()

	val, err := store.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "" {
		t.Errorf("expected empty string for nonexistent key, got '%s'", val)
	}
}

func TestMemoryChallengeStore_Expired(t *testing.T) {
	store := newTestStore()
	defer store.Close()
	ctx := context.Background()

	err := store.Put(ctx, "expiring-key", "expiring-value", 1*time.Nanosecond)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Wait for entry to expire
	time.Sleep(10 * time.Millisecond)

	val, err := store.Get(ctx, "expiring-key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "" {
		t.Errorf("expected empty string for expired key, got '%s'", val)
	}
}

func BenchmarkGenerateRegistrationChallenge(b *testing.B) {
	store := newTestStore()
	defer store.Close()
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		_, _ = generateRegistrationChallenge(ctx, store, "bench-user")
	}
}

func BenchmarkGenerateAssertionChallenge(b *testing.B) {
	store := newTestStore()
	defer store.Close()
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		_, _ = generateAssertionChallenge(ctx, store, "bench-user")
	}
}

func BenchmarkRandomBytes(b *testing.B) {
	buf := make([]byte, 32)
	for i := 0; i < b.N; i++ {
		_, _ = rand.Read(buf)
	}
}