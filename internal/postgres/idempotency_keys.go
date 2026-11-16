package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/LegionAlpha/valthera-backend/internal/idempotency_key"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IdempotencyKeyRepository struct {
	pool *pgxpool.Pool
}

func NewIdempotencyKeyRepository(pool *pgxpool.Pool) *IdempotencyKeyRepository {
	return &IdempotencyKeyRepository{
		pool: pool,
	}
}

var _ idempotency_key.IdempotencyKeyRepo = (*IdempotencyKeyRepository)(nil)

const lockTimeout = "2000ms"

func (r *IdempotencyKeyRepository) Claim(
	ctx context.Context,
	scope idempotency_key.Scope,
	fingerprint []byte,
) (existing *idempotency_key.Record, err error) {
	tx, ok := txFromContext(ctx)
	if !ok {
		return nil, errors.New("claim idempotency key: no transaction in context")
	}

	if _, err := tx.Exec(ctx, `SELECT set_config('lock_timeout', $1, true)`, lockTimeout); err != nil {
		return nil, fmt.Errorf("set lock timeout: %w", err)
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO idempotency_keys(user_id, operation, key, request_fingerprint)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, operation, key) DO NOTHING`,
		scope.UserID, scope.Operation, scope.Key, fingerprint)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "55P03" {
			return nil, idempotency_key.ErrInProgress
		}
		return nil, fmt.Errorf("insert idempotency key: %w", err)
	}

	if _, err := tx.Exec(ctx, `SELECT set_config('lock_timeout', '0', true)`); err != nil {
		return nil, fmt.Errorf("reset lock timeout: %w", err)
	}

	if tag.RowsAffected() == 1 {
		return nil, nil
	}

	rec := idempotency_key.Record{Scope: scope}
	err = tx.QueryRow(ctx, `
		SELECT request_fingerprint, resource_id, response_status, response_headers, response_body, created_at
		FROM idempotency_keys
		WHERE user_id = $1 AND operation = $2 AND key = $3
	`, scope.UserID, scope.Operation, scope.Key).Scan(&rec.Fingerprint, &rec.Result.ResourceID, &rec.Result.Status, &rec.Result.Headers, &rec.Result.Body, &rec.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("load idempotency key: %w", err)
	}

	return &rec, nil
}

func (r *IdempotencyKeyRepository) Complete(
	ctx context.Context,
	scope idempotency_key.Scope,
	result idempotency_key.Result,
) error {
	tx, ok := txFromContext(ctx)
	if !ok {
		return errors.New("complete idempotency key: no transaction in context")
	}

	tag, err := tx.Exec(ctx, `
		UPDATE idempotency_keys
		SET resource_id = $4, response_status = $5,
		    response_headers = $6, response_body = $7
		WHERE user_id = $1 AND operation = $2 AND key = $3`,
		scope.UserID, scope.Operation, scope.Key,
		result.ResourceID, result.Status, result.Headers, result.Body)
	if err != nil {
		return fmt.Errorf("complete idempotency key: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return errors.New("complete idempotency key: row not claimed")
	}
	return nil
}
