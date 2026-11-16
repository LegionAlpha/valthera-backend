package idempotency_key

const (
	MinKeyLength = 1
	MaxKeyLength = 255
)

func ValidateKey(raw string) (string, error) {
	if len(raw) < MinKeyLength || len(raw) > MaxKeyLength {
		return "", ErrInvalidKey
	}

	for i := 0; i < len(raw); i++ {
		if raw[i] < 0x21 || raw[i] > 0x7e {
			return "", ErrInvalidKey
		}
	}

	return raw, nil
}
