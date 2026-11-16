package idempotency_key

import (
	"time"

	"github.com/google/uuid"
)

type Scope struct {
	UserID    uuid.UUID
	Operation string
	Key       string
}

type Result struct {
	ResourceID uuid.UUID
	Status     int
	Headers    map[string]string
	Body       []byte
}

type Record struct {
	Scope       Scope
	Fingerprint []byte
	Result      Result
	CreatedAt   time.Time
}
