package identity

import (
	"context"
	"strings"
	"unicode/utf8"
)

const (
	MinPasswordLength = 8
	MaxPasswordLength = 128
)

func ValidatePassword(password string) error {
	if len(password) > MaxPasswordLength*utf8.UTFMax {
		return ErrInvalidPassword
	}

	if !utf8.ValidString(password) || strings.ContainsRune(password, '\x00') {
		return ErrInvalidPassword
	}

	length := utf8.RuneCountInString(password)
	if length < MinPasswordLength || length > MaxPasswordLength {
		return ErrInvalidPassword
	}

	return nil
}

type PasswordHasher interface {
	Hash(ctx context.Context, password string) (string, error)
	Verify(
		ctx context.Context,
		password string,
		encodedHash string,
	) (bool, error)
}
