package password

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"

	"github.com/LegionAlpha/valthera-backend/internal/identity"
)

const (
	memoryKiB   uint32 = 64 * 1024
	iterations  uint32 = 3
	parallelism uint8  = 1
	saltLength         = 16
	keyLength   uint32 = 32

	maxEncodedHashLength = 1024
)

var ErrInvalidHash = errors.New(
	"invalid or unsupported password hash",
)

type Argon2idHasher struct {
	slots chan struct{}
}

var _ identity.PasswordHasher = (*Argon2idHasher)(nil)

func NewArgon2idHasher(maxConcurrent int) (*Argon2idHasher, error) {
	if maxConcurrent < 1 {
		return nil, errors.New("maxConcurrent must be positive")
	}

	return &Argon2idHasher{
		slots: make(chan struct{}, maxConcurrent),
	}, nil
}

func (h *Argon2idHasher) acquire(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	select {
	case h.slots <- struct{}{}:
		if err := ctx.Err(); err != nil {
			<-h.slots
			return err
		}
		return nil

	default:
		return identity.ErrPasswordHashingBusy
	}
}

func (h *Argon2idHasher) release() {
	<-h.slots
}

func (h *Argon2idHasher) Hash(
	ctx context.Context,
	password string,
) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	if err := identity.ValidatePassword(password); err != nil {
		return "", err
	}

	if err := h.acquire(ctx); err != nil {
		return "", err
	}
	defer h.release()

	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}

	key := argon2.IDKey(
		[]byte(password),
		salt,
		iterations,
		memoryKiB,
		parallelism,
		keyLength,
	)

	if err := ctx.Err(); err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		memoryKiB,
		iterations,
		parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func decodeHash(encoded string) ([]byte, []byte, error) {
	if len(encoded) == 0 || len(encoded) > maxEncodedHashLength {
		return nil, nil, ErrInvalidHash
	}

	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return nil, nil, ErrInvalidHash
	}

	if parts[0] != "" ||
		parts[1] != "argon2id" ||
		parts[2] != fmt.Sprintf("v=%d", argon2.Version) {
		return nil, nil, ErrInvalidHash
	}

	expectedParams := fmt.Sprintf(
		"m=%d,t=%d,p=%d",
		memoryKiB,
		iterations,
		parallelism,
	)
	if parts[3] != expectedParams {
		return nil, nil, ErrInvalidHash
	}

	encoding := base64.RawStdEncoding.Strict()

	salt, err := encoding.DecodeString(parts[4])
	if err != nil || len(salt) != saltLength {
		return nil, nil, ErrInvalidHash
	}

	key, err := encoding.DecodeString(parts[5])
	if err != nil || len(key) != int(keyLength) {
		return nil, nil, ErrInvalidHash
	}

	if encoding.EncodeToString(salt) != parts[4] ||
		encoding.EncodeToString(key) != parts[5] {
		return nil, nil, ErrInvalidHash
	}

	return salt, key, nil
}

func (h *Argon2idHasher) Verify(
	ctx context.Context,
	password string,
	encodedHash string,
) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	if len(password) > identity.MaxPasswordLength*utf8.UTFMax ||
		!utf8.ValidString(password) ||
		strings.ContainsRune(password, '\x00') {
		return false, nil
	}

	if utf8.RuneCountInString(password) > identity.MaxPasswordLength {
		return false, nil
	}

	salt, expectedKey, err := decodeHash(encodedHash)
	if err != nil {
		return false, err
	}

	if err := h.acquire(ctx); err != nil {
		return false, err
	}
	defer h.release()

	actualKey := argon2.IDKey(
		[]byte(password),
		salt,
		iterations,
		memoryKiB,
		parallelism,
		keyLength,
	)

	if err := ctx.Err(); err != nil {
		return false, err
	}

	return subtle.ConstantTimeCompare(actualKey, expectedKey) == 1, nil
}
