package idempotency_key

import "errors"

var (
	ErrKeyReused  = errors.New("key reused")
	ErrInProgress = errors.New("operation in progress")
	ErrInvalidKey = errors.New("wrong key passed")
)
