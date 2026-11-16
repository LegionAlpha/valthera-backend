package user_session

import (
	"time"

	"github.com/google/uuid"
)

type UserSession struct {
	Id         uuid.UUID
	UserId     uuid.UUID
	TokenHash  []byte
	CreatedAt  *time.Time
	LastSeenAt *time.Time
	ExpiresAt  *time.Time
	RevokedAt  *time.Time
}
