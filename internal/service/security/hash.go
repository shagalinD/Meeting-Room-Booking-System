package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type Argon2idConfig struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

var defaultArgon2idConfig = Argon2idConfig{
	Memory:      64 * 1024, // 64 MB
	Iterations:  3,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}

func generateRandomBytes(n uint32) ([]byte, error) {
	bytes := make([]byte, n)
	_, err := rand.Read(bytes)
	return bytes, err
}

func HashPassword(password string) (string, error) {
	config := defaultArgon2idConfig
	salt, err := generateRandomBytes(config.SaltLength)

	if err != nil {
		return "", err
	}

	hash := argon2.IDKey(
		[]byte(password), 
		salt,
	 	config.Iterations, 
		config.Memory, 
		config.Parallelism, 
		config.SaltLength,
	)

	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	finalHash := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", config.Memory, config.Iterations, config.Parallelism, encodedSalt, encodedHash)

	return finalHash, nil
}

func VerifyPassword(password, encodedHash string) (bool, error) {
	config, hash, salt, err := decodeHash(encodedHash)

	if err != nil {
		return false, err 
	}

	passwordHash := argon2.IDKey(
		[]byte(password), 
		salt, 
		config.Iterations,
		config.Memory,
		config.Parallelism,
		config.KeyLength,
	)

	if subtle.ConstantTimeCompare(passwordHash, hash) == 1 {
		return true, nil 
	}

	return false, nil
}

func decodeHash(hash string) (Argon2idConfig, []byte, []byte, error) {
	parts := strings.Split(hash, "$")

	if len(parts) != 6 {
		return Argon2idConfig{}, nil, nil, errors.New("invalid hash format")
	}

	if parts[0] != "argon2id" {
		
	}

	var version int
	var memory, iterations uint32 
	var parallelism uint8 
    _, err := fmt.Sscanf(parts[2], "v=%d", &version)
    if err != nil {
        return Argon2idConfig{}, nil, nil, err
    }
    
    _, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
    if err != nil {
        return Argon2idConfig{}, nil, nil, err
    }
    
    // Декодируем соль и хеш
    salt, err := base64.RawStdEncoding.DecodeString(parts[4])
    if err != nil {
        return Argon2idConfig{}, nil, nil, err
    }
    
    resultHash, err := base64.RawStdEncoding.DecodeString(parts[5])
    if err != nil {
        return Argon2idConfig{}, nil, nil, err
    }
    
    params := Argon2idConfig{
        Memory:      memory,
        Iterations:  iterations,
        Parallelism: parallelism,
        SaltLength:  uint32(len(salt)),
        KeyLength:   uint32(len(hash)),
    }
    
    return params, salt, resultHash, nil
}