package service

import (
	"crypto/rand"
	"testing"
)

func TestGenerateRegistrationChallenge(t *testing.T) {
	challenge, err := generateRegistrationChallenge("test-user-id")
	if err != nil {
		t.Fatalf("generateRegistrationChallenge failed: %v", err)
	}

	if len(challenge) != 32 {
		t.Errorf("expected 32-byte challenge, got %d bytes", len(challenge))
	}
}

func TestGenerateRegistrationChallengeUniqueness(t *testing.T) {
	ch1, _ := generateRegistrationChallenge("user-1")
	ch2, _ := generateRegistrationChallenge("user-1")

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
	challenge, err := generateAssertionChallenge("test-user-id")
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
	ch1, _ := generateAssertionChallenge("user-1")
	ch2, _ := generateAssertionChallenge("user-1")

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

func BenchmarkGenerateRegistrationChallenge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = generateRegistrationChallenge("bench-user")
	}
}

func BenchmarkGenerateAssertionChallenge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = generateAssertionChallenge("bench-user")
	}
}

func BenchmarkRandomBytes(b *testing.B) {
	buf := make([]byte, 32)
	for i := 0; i < b.N; i++ {
		_, _ = rand.Read(buf)
	}
}