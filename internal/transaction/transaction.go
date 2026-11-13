package transaction

import (
	"time"

	"github.com/google/uuid"
)

type Type string

const (
	TypeIncome   Type = "income"
	TypeExpense  Type = "expense"
	TypeTransfer Type = "transfer"
)

type Transaction struct {
	ID                   uuid.UUID
	UserID               uuid.UUID
	Type                 Type
	AccountID            uuid.UUID
	DestinationAccountID *uuid.UUID
	CategoryID           *uuid.UUID
	AmountMinor          int64
	Date                 time.Time
	Comment              string
	Version              int64
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
