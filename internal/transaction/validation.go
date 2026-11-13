package transaction

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"golang.org/x/text/unicode/norm"
)

const (
	MinAmountMinor int64 = 1
	MaxAmountMinor int64 = 10_000_000_000_000
)

var (
	ErrInvalidType           = errors.New("invalid transaction type")
	ErrAmountOutOfRange      = errors.New("amount is out of range")
	ErrInvalidDate           = errors.New("invalid transaction date")
	ErrInvalidComment        = errors.New("invalid transaction comment")
	ErrInvalidAccountID      = errors.New("invalid account ID")
	ErrInvalidCategoryID     = errors.New("invalid category ID")
	ErrCategoryRequired      = errors.New("category is required")
	ErrUnexpectedCategory    = errors.New("transfer must not have a category")
	ErrDestinationRequired   = errors.New("transfer requires a destination account")
	ErrUnexpectedDestination = errors.New(
		"only a transfer can have a destination account",
	)
	ErrSameAccountTransfer = errors.New(
		"transfer source and destination must differ",
	)
)

func ParseType(raw string) (Type, error) {
	switch value := Type(raw); value {
	case TypeIncome, TypeExpense, TypeTransfer:
		return value, nil
	default:
		return "", ErrInvalidType
	}
}

func ValidateAmount(amountMinor int64) error {
	if amountMinor < MinAmountMinor || amountMinor > MaxAmountMinor {
		return ErrAmountOutOfRange
	}

	return nil
}

// NormalizeDate keeps only the calendar date: transactions have no time
// of day and no time zone.
func NormalizeDate(date time.Time) (time.Time, error) {
	if date.IsZero() {
		return time.Time{}, ErrInvalidDate
	}

	year, month, day := date.Date()

	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC), nil
}

func NormalizeComment(raw string) (string, error) {
	if !utf8.ValidString(raw) || strings.ContainsRune(raw, '\x00') {
		return "", ErrInvalidComment
	}

	return norm.NFC.String(strings.TrimSpace(raw)), nil
}

// ValidateShape checks which references a transaction of the given type
// must and must not have.
func ValidateShape(
	transactionType Type,
	accountID uuid.UUID,
	destinationAccountID *uuid.UUID,
	categoryID *uuid.UUID,
) error {
	if accountID == uuid.Nil {
		return ErrInvalidAccountID
	}

	if destinationAccountID != nil && *destinationAccountID == uuid.Nil {
		return ErrInvalidAccountID
	}

	if categoryID != nil && *categoryID == uuid.Nil {
		return ErrInvalidCategoryID
	}

	if transactionType == TypeTransfer {
		if destinationAccountID == nil {
			return ErrDestinationRequired
		}
		if *destinationAccountID == accountID {
			return ErrSameAccountTransfer
		}
		if categoryID != nil {
			return ErrUnexpectedCategory
		}

		return nil
	}

	if destinationAccountID != nil {
		return ErrUnexpectedDestination
	}
	if categoryID == nil {
		return ErrCategoryRequired
	}

	return nil
}
