package identity

import (
	"net/mail"
	"strings"
	"unicode/utf8"
)

const MaxEmailLength = 254

func NormalizeEmail(raw string) (string, error) {
	if !utf8.ValidString(raw) {
		return "", ErrInvalidEmail
	}

	email := strings.TrimSpace(raw)

	if len(email) == 0 || len(email) > MaxEmailLength {
		return "", ErrInvalidEmail
	}

	for i := 0; i < len(email); i++ {
		if email[i] <= ' ' || email[i] >= 127 {
			return "", ErrInvalidEmail
		}
	}

	if strings.ContainsAny(email, "\"()<>") {
		return "", ErrInvalidEmail
	}

	address, err := mail.ParseAddress(email)
	if err != nil {
		return "", ErrInvalidEmail
	}

	if address.Name != "" || address.Address != email {
		return "", ErrInvalidEmail
	}

	return strings.ToLower(email), nil
}
