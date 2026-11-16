package user_session

import (
	"context"

	"github.com/google/uuid"
)

type UserSessionRepo interface {
	Create(ctx context.Context, user_id uuid.UUID, tokenHash []byte) (UserSession, error)
	Get(ctx context.Context, user_id uuid.UUID, sessionToken []byte) error
	Delete(ctx context.Context, user_id uuid.UUID) error
}
