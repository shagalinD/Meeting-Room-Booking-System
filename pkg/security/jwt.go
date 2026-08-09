package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// CustomClaims с полным набором полей
type CustomClaims struct {
    UserID    string   `json:"user_id"`
    Role      string `json:"role"`
    
    // Стандартные claims
    jwt.RegisteredClaims
}

func CreateAccessToken(userId string, role string, secret []byte) (string, error) {
	// 1. Создаем claims со всеми полями
	claims := &CustomClaims{
			UserID:   userId,
			Role:     role,
			RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
					NotBefore: jwt.NewNumericDate(time.Now()),
			},
	}

	// 2. Создаем токен
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	
	// 3. Подписываем токен
	tokenString, err := token.SignedString(secret)
	if err != nil {
			return "", err
	}

	return tokenString, nil
}

func CreateRefreshToken(userId string, role string, secret []byte) (string, error) {
	// 1. Создаем claims со всеми полями
	claims := &CustomClaims{
			UserID:   userId,
			Role:     role,
			RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
					NotBefore: jwt.NewNumericDate(time.Now()),
			},
	}

	// 2. Создаем токен
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	
	// 3. Подписываем токен
	tokenString, err := token.SignedString(secret)
	if err != nil {
			return "", err
	}

	return tokenString, nil
}

func ParseToken(tokenString string, secret []byte) (*CustomClaims, error) {
    // 4. Парсим токен обратно
    parsedToken, err := jwt.ParseWithClaims(
        tokenString,
        &CustomClaims{},
        func(token *jwt.Token) (any, error) {
            // Проверяем алгоритм
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
            }
            return secret, nil
        },
    )
    
		if err != nil {
			return nil, err
		}
    
    if !parsedToken.Valid {
        return nil, errors.New("token is invalid")
    }
    
    claims, ok := parsedToken.Claims.(*CustomClaims)
    if !ok {
        return nil, errors.New("invalid claims type")
    }
    
    return claims, nil
}

