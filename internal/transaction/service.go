package transaction

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type CreateInput struct {
	Type                 string
	AccountID            uuid.UUID
	DestinationAccountID *uuid.UUID
	CategoryID           *uuid.UUID
	AmountMinor          int64
	Date                 time.Time
	Comment              string
}

type CreateParams struct {
	UserID               uuid.UUID
	Type                 Type
	AccountID            uuid.UUID
	DestinationAccountID *uuid.UUID
	CategoryID           *uuid.UUID
	AmountMinor          int64
	Date                 time.Time
	Comment              string
}

type Creator interface {
	Create(ctx context.Context, params CreateParams) (Transaction, error)
}

type Reader interface {
	GetByID(
		ctx context.Context,
		userID uuid.UUID,
		transactionID uuid.UUID,
	) (Transaction, error)
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
) (Transaction, error) {
	if userID == uuid.Nil {
		return Transaction{}, ErrInvalidUserID
	}

	transactionType, err := ParseType(input.Type)
	if err != nil {
		return Transaction{}, err
	}

	if err := ValidateShape(
		transactionType,
		input.AccountID,
		input.DestinationAccountID,
		input.CategoryID,
	); err != nil {
		return Transaction{}, err
	}

	if err := ValidateAmount(input.AmountMinor); err != nil {
		return Transaction{}, err
	}

	date, err := NormalizeDate(input.Date)
	if err != nil {
		return Transaction{}, err
	}

	comment, err := NormalizeComment(input.Comment)
	if err != nil {
		return Transaction{}, err
	}

	created, err := s.creator.Create(ctx, CreateParams{
		UserID:               userID,
		Type:                 transactionType,
		AccountID:            input.AccountID,
		DestinationAccountID: input.DestinationAccountID,
		CategoryID:           input.CategoryID,
		AmountMinor:          input.AmountMinor,
		Date:                 date,
		Comment:              comment,
	})
	if err != nil {
		return Transaction{}, fmt.Errorf("create transaction: %w", err)
	}

	return created, nil
}

func (s *Service) GetByID(
	ctx context.Context,
	userID uuid.UUID,
	transactionID uuid.UUID,
) (Transaction, error) {
	if userID == uuid.Nil {
		return Transaction{}, ErrInvalidUserID
	}

	if transactionID == uuid.Nil {
		return Transaction{}, ErrInvalidTransactionID
	}

	found, err := s.reader.GetByID(ctx, userID, transactionID)
	if err != nil {
		return Transaction{}, fmt.Errorf("get transaction: %w", err)
	}

	return found, nil
}
