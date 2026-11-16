package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/LegionAlpha/valthera-backend/internal/user_session"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserSessionRepository struct {
	pool *pgxpool.Pool
}

func NewUserSessionRepository(pool *pgxpool.Pool) *UserSessionRepository {
	return &UserSessionRepository{
		pool: pool,
	}
}

var _ user_session.UserSessionRepo = (*UserSessionRepository)(nil)

func (r *UserSessionRepository) Create(ctx context.Context, userId uuid.UUID, tokenHash []byte) (user_session.UserSession, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted,
	})
	if err != nil {
		return user_session.UserSession{}, fmt.Errorf(
			"begin user session creation: %w",
			err,
		)
	}

	defer func() {
		rollbackCtx, rollbackCancel := context.WithTimeout(
			context.Background(),
			2*time.Second,
		)
		defer rollbackCancel()

		_ = tx.Rollback(rollbackCtx)
	}()

	var created user_session.UserSession

	err = tx.QueryRow(ctx, `
		INSERT INTO sessions (
			user_id,
			token_hash,
			created_at,
			last_seen_at,
			expires_at,
		) VALUES ($1, $2, $3, $4, $5)

	`, userId.String(), tokenHash, time.Now(), time.Now(), time.Now()).Scan(
		&created.Id,
		&created.UserId,
		&created.TokenHash,
		&created.CreatedAt,
		&created.LastSeenAt,
		&created.ExpiresAt,
		&created.RevokedAt,
	)

	if err != nil {
		return user_session.UserSession{}, fmt.Errorf(
			"insert user session: %w", err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return user_session.UserSession{}, fmt.Errorf(
			"commit user session creation: %w",
			err,
		)
	}

	// TODO: отловить ошибки

	return created, nil
}

func (r *UserSessionRepository) Get(ctx context.Context, user_id uuid.UUID, sessionToken []byte) error {
	panic("NOT IMPLEMENTED YET")
}

func (r *UserSessionRepository) Delete(ctx context.Context, user_id uuid.UUID) error {
	panic("NOT IMPLEMENTED YET")
}
