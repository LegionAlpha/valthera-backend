package account

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type CreateInput struct {
	Name                string
	InitialBalanceMinor int64
}

type CreateParams struct {
	UserID              uuid.UUID
	Name                string
	InitialBalanceMinor int64
}

type Creator interface {
	Create(ctx context.Context, params CreateParams) (Account, error)
}

type Service struct {
	creator Creator
}

func NewService(creator Creator) *Service {
	return &Service{
		creator: creator,
	}
}

func (s *Service) Create(
	ctx context.Context,
	userID uuid.UUID,
	input CreateInput,
) (Account, error) {
	if userID == uuid.Nil {
		return Account{}, ErrInvalidUserID
	}
	name, err := NormalizeName(input.Name)
	if err != nil {
		return Account{}, err
	}

	if err := ValidateInitialBalance(input.InitialBalanceMinor); err != nil {
		return Account{}, err
	}

	created, err := s.creator.Create(ctx, CreateParams{
		UserID:              userID,
		Name:                name,
		InitialBalanceMinor: input.InitialBalanceMinor,
	})
	if err != nil {
		return Account{}, fmt.Errorf("create account: %w", err)
	}

	return created, nil
}
