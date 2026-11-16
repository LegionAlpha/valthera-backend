package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/LegionAlpha/valthera-backend/internal/idempotency_key"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txKey struct{}

type TxRunnerRepository struct {
	pool *pgxpool.Pool
}

func NewTxRunnerRepository(pool *pgxpool.Pool) *TxRunnerRepository {
	return &TxRunnerRepository{
		pool: pool,
	}
}

var _ idempotency_key.TxRunnerRepo = (*TxRunnerRepository)(nil)

func txFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	return tx, ok
}

func (r *TxRunnerRepository) Run(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := txFromContext(ctx); ok {
		return fn(ctx)
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		rbCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = tx.Rollback(rbCtx)
	}()

	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
