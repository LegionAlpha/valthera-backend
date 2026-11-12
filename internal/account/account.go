package account

import (
	"time"

	"github.com/google/uuid"
)

type Account struct {
	ID                  uuid.UUID
	UserID              uuid.UUID
	Name                string
	InitialBalanceMinor int64
	Version             int64
	ArchivedAt          *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (a Account) IsArchived() bool {
	return a.ArchivedAt != nil
}
