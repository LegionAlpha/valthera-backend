package account

import (
	"errors"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

const (
	MaxNameLength = 80

	MinInitialBalanceMinor int64 = -10_000_000_000_000
	MaxInitialBalanceMinor int64 = 10_000_000_000_000
)

var (
	ErrInvalidName              = errors.New("invalid account name")
	ErrInitialBalanceOutOfRange = errors.New(
		"initial balance is out of range",
	)
)

func NormalizeName(raw string) (string, error) {
	if !utf8.ValidString(raw) || strings.ContainsRune(raw, '\x00') {
		return "", ErrInvalidName
	}

	name := norm.NFC.String(strings.TrimSpace(raw))

	length := utf8.RuneCountInString(name)
	if length < 1 || length > MaxNameLength {
		return "", ErrInvalidName
	}

	return name, nil
}

func ValidateInitialBalance(amountMinor int64) error {
	if amountMinor < MinInitialBalanceMinor ||
		amountMinor > MaxInitialBalanceMinor {
		return ErrInitialBalanceOutOfRange
	}

	return nil
}
