// security_test.go
package security

import (
	"testing"
)

func TestTokenLifecycle(t *testing.T) {
    secret := []byte("my_fixed_secret_1234567890")
    userID := "user-123"
    role := "admin"
    
    // 1. Создаем токен
    token, err := CreateAccessToken(userID, role, secret)
    if err != nil {
        t.Fatalf("Create failed: %v", err)
    }
    t.Logf("Token: %s", token)
    
    // 2. Проверяем токен
    claims, err := ParseToken(token, secret)
    if err != nil {
        t.Fatalf("Parse failed: %v", err)
    }
    
    // 3. Проверяем данные
    if claims.UserID != userID {
        t.Errorf("UserID mismatch: got %s, want %s", claims.UserID, userID)
    }
    if claims.Role != role {
        t.Errorf("Role mismatch: got %s, want %s", claims.Role, role)
    }
}