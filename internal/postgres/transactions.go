package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/LegionAlpha/valthera-backend/internal/transaction"
	"github.com/google/uuid"
)

type TransactionRepository struct {
	pool     *pgxpool.Pool
	txRunner *TxRunnerRepository
}

func NewTransactionRepository(
	pool *pgxpool.Pool,
	txRunner *TxRunnerRepository,
) *TransactionRepository {
	return &TransactionRepository{
		pool:     pool,
		txRunner: txRunner,
	}
}

var _ transaction.TransactionRepo = (*TransactionRepository)(nil)

const transactionColumns = `
	id,
	user_id,
	type,
	account_id,
	destination_account_id,
	category_id,
	amount_minor,
	date,
	comment,
	version,
	created_at,
	updated_at
`

func (r *TransactionRepository) Create(
	ctx context.Context,
	params transaction.CreateParams,
) (transaction.Transaction, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var created transaction.Transaction

	err := r.txRunner.Run(ctx, func(ctx context.Context) error {
		tx, ok := txFromContext(ctx)
		if !ok {
			return errors.New("create transaction: no transaction in context")
		}

		if err := lockActiveAccount(
			ctx, tx, params.UserID, params.AccountID,
		); err != nil {
			return err
		}

		if params.DestinationAccountID != nil {
			if err := lockActiveAccount(
				ctx, tx, params.UserID, *params.DestinationAccountID,
			); err != nil {
				return err
			}
		}

		var err error
		created, err = scanTransaction(tx.QueryRow(ctx, `
			INSERT INTO transactions (
				user_id,
				type,
				account_id,
				destination_account_id,
				category_id,
				amount_minor,
				date,
				comment
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING`+transactionColumns,
			params.UserID,
			string(params.Type),
			params.AccountID,
			params.DestinationAccountID,
			params.CategoryID,
			params.AmountMinor,
			params.Date,
			params.Comment,
		))
		if err != nil {
			return fmt.Errorf("insert transaction: %w", err)
		}

		return nil
	})
	if err != nil {
		return transaction.Transaction{}, err
	}

	return created, nil
}

func (r *TransactionRepository) GetByID(
	ctx context.Context,
	userID uuid.UUID,
	transactionID uuid.UUID,
) (transaction.Transaction, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	found, err := scanTransaction(r.pool.QueryRow(ctx, `
		SELECT`+transactionColumns+`
		FROM transactions
		WHERE id = $1 AND user_id = $2
	`, transactionID, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return transaction.Transaction{}, transaction.ErrNotFound
	}
	if err != nil {
		return transaction.Transaction{}, fmt.Errorf(
			"select transaction: %w",
			err,
		)
	}

	return found, nil
}

// lockActiveAccount checks that the account belongs to the user and is
// not archived. FOR SHARE keeps it from being archived concurrently.
func lockActiveAccount(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	accountID uuid.UUID,
) error {
	var archivedAt *time.Time

	err := tx.QueryRow(ctx, `
		SELECT archived_at
		FROM accounts
		WHERE id = $1 AND user_id = $2
		FOR SHARE
	`, accountID, userID).Scan(&archivedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return transaction.ErrAccountNotFound
	}
	if err != nil {
		return fmt.Errorf("lock transaction account: %w", err)
	}

	if archivedAt != nil {
		return transaction.ErrAccountArchived
	}

	return nil
}

func scanTransaction(row pgx.Row) (transaction.Transaction, error) {
	var (
		found           transaction.Transaction
		transactionType string
	)

	err := row.Scan(
		&found.ID,
		&found.UserID,
		&transactionType,
		&found.AccountID,
		&found.DestinationAccountID,
		&found.CategoryID,
		&found.AmountMinor,
		&found.Date,
		&found.Comment,
		&found.Version,
		&found.CreatedAt,
		&found.UpdatedAt,
	)
	if err != nil {
		return transaction.Transaction{}, err
	}

	found.Type = transaction.Type(transactionType)

	return found, nil
}
