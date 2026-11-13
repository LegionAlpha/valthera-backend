package transaction_test

import (
	"errors"
	"testing"
	"time"

	"github.com/LegionAlpha/valthera-backend/internal/transaction"
	"github.com/google/uuid"
)

func TestParseType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    transaction.Type
		wantErr error
	}{
		{name: "income", input: "income", want: transaction.TypeIncome},
		{name: "expense", input: "expense", want: transaction.TypeExpense},
		{name: "transfer", input: "transfer", want: transaction.TypeTransfer},
		{name: "empty", input: "", wantErr: transaction.ErrInvalidType},
		{name: "wrong case", input: "Income", wantErr: transaction.ErrInvalidType},
		{name: "unknown", input: "refund", wantErr: transaction.ErrInvalidType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transaction.ParseType(tt.input)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}

			if got != tt.want {
				t.Errorf("result = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateAmount(t *testing.T) {
	tests := []struct {
		name    string
		input   int64
		wantErr error
	}{
		{name: "minimum", input: 1},
		{name: "typical", input: 125050},
		{name: "maximum", input: 10_000_000_000_000},
		{
			name:    "zero",
			input:   0,
			wantErr: transaction.ErrAmountOutOfRange,
		},
		{
			name:    "negative",
			input:   -125050,
			wantErr: transaction.ErrAmountOutOfRange,
		},
		{
			name:    "above maximum",
			input:   10_000_000_000_001,
			wantErr: transaction.ErrAmountOutOfRange,
		},
		{
			name:    "maximum int64",
			input:   1<<63 - 1,
			wantErr: transaction.ErrAmountOutOfRange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := transaction.ValidateAmount(tt.input)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeDate(t *testing.T) {
	moscow := time.FixedZone("MSK", 3*60*60)

	got, err := transaction.NormalizeDate(
		time.Date(2026, time.November, 12, 1, 30, 0, 0, moscow),
	)
	if err != nil {
		t.Fatalf("NormalizeDate() error = %v", err)
	}

	want := time.Date(2026, time.November, 12, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) || got.Location() != time.UTC {
		t.Errorf("date = %v, want %v", got, want)
	}

	if _, err := transaction.NormalizeDate(time.Time{}); !errors.Is(
		err,
		transaction.ErrInvalidDate,
	) {
		t.Errorf("zero date error = %v, want %v", err, transaction.ErrInvalidDate)
	}
}

func TestNormalizeComment(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{name: "empty", input: "", want: ""},
		{name: "trim whitespace", input: "  кофе \n", want: "кофе"},
		{name: "normalize NFC", input: "ёж", want: "ёж"},
		{
			name:    "invalid UTF-8",
			input:   string([]byte{0xff}),
			wantErr: transaction.ErrInvalidComment,
		},
		{
			name:    "null character",
			input:   "кофе\x00",
			wantErr: transaction.ErrInvalidComment,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transaction.NormalizeComment(tt.input)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}

			if got != tt.want {
				t.Errorf("result = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateShape(t *testing.T) {
	accountID := uuid.MustParse("8b0e0b8e-e665-42e7-87b9-2f10a228c951")
	otherAccountID := uuid.MustParse("5f7c2a1e-3b0d-4c8e-9a6f-1d2e3f4a5b6c")
	categoryID := uuid.MustParse("0c9d8e7f-6a5b-4c3d-8e1f-0a9b8c7d6e5f")
	nilID := uuid.Nil

	tests := []struct {
		name        string
		kind        transaction.Type
		account     uuid.UUID
		destination *uuid.UUID
		category    *uuid.UUID
		wantErr     error
	}{
		{
			name:     "expense with category",
			kind:     transaction.TypeExpense,
			account:  accountID,
			category: &categoryID,
		},
		{
			name:     "income with category",
			kind:     transaction.TypeIncome,
			account:  accountID,
			category: &categoryID,
		},
		{
			name:        "transfer between accounts",
			kind:        transaction.TypeTransfer,
			account:     accountID,
			destination: &otherAccountID,
		},
		{
			name:     "missing account",
			kind:     transaction.TypeExpense,
			account:  uuid.Nil,
			category: &categoryID,
			wantErr:  transaction.ErrInvalidAccountID,
		},
		{
			name:     "nil category",
			kind:     transaction.TypeExpense,
			account:  accountID,
			category: &nilID,
			wantErr:  transaction.ErrInvalidCategoryID,
		},
		{
			name:    "expense without category",
			kind:    transaction.TypeExpense,
			account: accountID,
			wantErr: transaction.ErrCategoryRequired,
		},
		{
			name:        "expense with destination",
			kind:        transaction.TypeExpense,
			account:     accountID,
			destination: &otherAccountID,
			category:    &categoryID,
			wantErr:     transaction.ErrUnexpectedDestination,
		},
		{
			name:    "transfer without destination",
			kind:    transaction.TypeTransfer,
			account: accountID,
			wantErr: transaction.ErrDestinationRequired,
		},
		{
			name:        "transfer to the same account",
			kind:        transaction.TypeTransfer,
			account:     accountID,
			destination: &accountID,
			wantErr:     transaction.ErrSameAccountTransfer,
		},
		{
			name:        "transfer with category",
			kind:        transaction.TypeTransfer,
			account:     accountID,
			destination: &otherAccountID,
			category:    &categoryID,
			wantErr:     transaction.ErrUnexpectedCategory,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := transaction.ValidateShape(
				tt.kind,
				tt.account,
				tt.destination,
				tt.category,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
