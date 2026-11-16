package transaction

import (
	"context"

	"github.com/google/uuid"
)

type TransactionRepo interface {
	Create(ctx context.Context, params CreateParams) (Transaction, error)
	GetByID(
		ctx context.Context,
		userID uuid.UUID,
		transactionID uuid.UUID,
	) (Transaction, error)
}
