package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/LegionAlpha/valthera-backend/internal/account"
	"github.com/google/uuid"
)

type AccountRepository struct {
	pool *pgxpool.Pool
}

func NewAccountRepository(pool *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{
		pool: pool,
	}
}

var _ account.Creator = (*AccountRepository)(nil)

func (r *AccountRepository) Create(
	ctx context.Context,
	params account.CreateParams,
) (account.Account, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted,
	})
	if err != nil {
		return account.Account{}, fmt.Errorf(
			"begin account creation: %w",
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

	var userID uuid.UUID

	err = tx.QueryRow(ctx, `
		SELECT id
		FROM users
		WHERE id = $1
		FOR UPDATE
	`, params.UserID).Scan(&userID)

	if errors.Is(err, pgx.ErrNoRows) {
		return account.Account{}, account.ErrUserNotFound
	}
	if err != nil {
		return account.Account{}, fmt.Errorf(
			"lock account owner: %w",
			err,
		)
	}

	var created account.Account

	err = tx.QueryRow(ctx, `
		INSERT INTO accounts (
			user_id,
			name,
			initial_balance_minor
		)
		VALUES ($1, $2, $3)
		RETURNING
			id,
			user_id,
			name,
			initial_balance_minor,
			version,
			archived_at,
			created_at,
			updated_at
	`,
		params.UserID,
		params.Name,
		params.InitialBalanceMinor,
	).Scan(
		&created.ID,
		&created.UserID,
		&created.Name,
		&created.InitialBalanceMinor,
		&created.Version,
		&created.ArchivedAt,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return account.Account{}, fmt.Errorf(
			"insert account: %w",
			err,
		)
	}
	var aggregateOK bool

	err = tx.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(initial_balance_minor), 0)
			BETWEEN -9007199254740991 AND 9007199254740991
		FROM accounts
		WHERE user_id = $1
	`, params.UserID).Scan(&aggregateOK)

	if err != nil {
		return account.Account{}, fmt.Errorf(
			"check account aggregate: %w",
			err,
		)
	}

	if !aggregateOK {
		return account.Account{}, account.ErrAggregateOutOfRange
	}
	if err := tx.Commit(ctx); err != nil {
		return account.Account{}, fmt.Errorf(
			"commit account creation: %w",
			err,
		)
	}

	return created, nil
}
