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

type Reader interface {
	GetByID(
		ctx context.Context,
		userID uuid.UUID,
		accountID uuid.UUID,
	) (Account, error)
}

type Service struct {
	creator Creator
	reader  Reader
}

func NewService(creator Creator, reader Reader) *Service {
	return &Service{
		creator: creator,
		reader:  reader,
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

func (s *Service) GetByID(
	ctx context.Context,
	userID uuid.UUID,
	accountID uuid.UUID,
) (Account, error) {
	if userID == uuid.Nil {
		return Account{}, ErrInvalidUserID
	}

	if accountID == uuid.Nil {
		return Account{}, ErrInvalidAccountID
	}

	found, err := s.reader.GetByID(ctx, userID, accountID)
	if err != nil {
		return Account{}, fmt.Errorf("get account: %w", err)
	}

	return found, nil
}
