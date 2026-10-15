package security

import (
	"strings"
	"testing"
)

func TestHashPasswordAndVerifyPassword(t *testing.T) {
	const password = "correct-horse-battery-staple"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("HashPassword() = %q, want Argon2id encoded hash", hash)
	}

	match, err := VerifyPassword(password, hash)
	if err != nil || !match {
		t.Fatalf("VerifyPassword(correct password) = (%v, %v), want (true, nil)", match, err)
	}
	match, err = VerifyPassword("incorrect-password", hash)
	if err != nil || match {
		t.Fatalf("VerifyPassword(incorrect password) = (%v, %v), want (false, nil)", match, err)
	}
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	match, err := VerifyPassword("password", "not-a-valid-encoded-hash")
	if err == nil || match {
		t.Fatalf("VerifyPassword() = (%v, %v), want (false, error)", match, err)
	}
}
