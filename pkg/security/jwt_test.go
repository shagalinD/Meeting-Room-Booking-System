// security_test.go
package security

import (
	"testing"
)

func TestTokenLifecycle(t *testing.T) {
	secret := []byte("my_fixed_secret_1234567890")
	userID := "user-123"
	role := "admin"

	token, err := CreateAccessToken(userID, role, secret)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	claims, err := ParseToken(token, secret)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("UserID mismatch: got %s, want %s", claims.UserID, userID)
	}
	if claims.Role != role {
		t.Errorf("Role mismatch: got %s, want %s", claims.Role, role)
	}
}

func TestParseTokenRejectsWrongSecretAndMalformedToken(t *testing.T) {
	token, err := CreateAccessToken("user-123", "admin", []byte("original-secret"))
	if err != nil {
		t.Fatalf("CreateAccessToken() error = %v", err)
	}

	if _, err := ParseToken(token, []byte("different-secret")); err == nil {
		t.Fatal("ParseToken() accepted a token signed with a different secret")
	}
	if _, err := ParseToken("not-a-jwt", []byte("original-secret")); err == nil {
		t.Fatal("ParseToken() accepted a malformed token")
	}
}
