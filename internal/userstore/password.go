package userstore

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const (
	// BcryptCost defines the password hashing cost.
	BcryptCost = 12
)

var (
	ErrEmptyPassword = errors.New("password is required")
)

// HashPassword hashes plain text password using bcrypt.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrEmptyPassword
	}

	raw, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// VerifyPassword compares a bcrypt hash and plain text password.
func VerifyPassword(hash, password string) error {
	if hash == "" || password == "" {
		return ErrEmptyPassword
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
