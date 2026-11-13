package transaction_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LegionAlpha/valthera-backend/internal/transaction"
	"github.com/google/uuid"
)

type creatorFunc func(
	context.Context,
	transaction.CreateParams,
) (transaction.Transaction, error)

func (f creatorFunc) Create(
	ctx context.Context,
	params transaction.CreateParams,
) (transaction.Transaction, error) {
	return f(ctx, params)
}

type unexpectedReader struct {
	t *testing.T
}

func (r unexpectedReader) GetByID(
	context.Context,
	uuid.UUID,
	uuid.UUID,
) (transaction.Transaction, error) {
	r.t.Helper()
	r.t.Fatal("reader must not be called during transaction creation")
	return transaction.Transaction{}, nil
}

var (
	testUserID    = uuid.MustParse("d7386164-7cc2-4aa3-a51c-338da77e57a6")
	testAccountID = uuid.MustParse("8b0e0b8e-e665-42e7-87b9-2f10a228c951")
)

func TestServiceCreate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	categoryID := uuid.MustParse("0c9d8e7f-6a5b-4c3d-8e1f-0a9b8c7d6e5f")
	date := time.Date(2026, time.November, 12, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, time.November, 12, 10, 0, 0, 0, time.UTC)

	want := transaction.Transaction{
		ID:          uuid.MustParse("5f7c2a1e-3b0d-4c8e-9a6f-1d2e3f4a5b6c"),
		UserID:      testUserID,
		Type:        transaction.TypeExpense,
		AccountID:   testAccountID,
		CategoryID:  &categoryID,
		AmountMinor: 125050,
		Date:        date,
		Comment:     "Кофе",
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	calls := 0

	creator := creatorFunc(func(
		gotCtx context.Context,
		params transaction.CreateParams,
	) (transaction.Transaction, error) {
		calls++

		if gotCtx != ctx {
			t.Error("request context was not forwarded")
		}

		wantParams := transaction.CreateParams{
			UserID:      testUserID,
			Type:        transaction.TypeExpense,
			AccountID:   testAccountID,
			CategoryID:  &categoryID,
			AmountMinor: 125050,
			Date:        date,
			Comment:     "Кофе",
		}

		if params != wantParams {
			t.Errorf("params = %+v, want %+v", params, wantParams)
		}

		return want, nil
	})

	service := transaction.NewService(
		creator,
		unexpectedReader{t: t},
	)
	got, err := service.Create(ctx, testUserID, transaction.CreateInput{
		Type:        "expense",
		AccountID:   testAccountID,
		CategoryID:  &categoryID,
		AmountMinor: 125050,
		Date:        date,
		Comment:     "  Кофе  ",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if calls != 1 {
		t.Errorf("storage calls = %d, want 1", calls)
	}

	if got != want {
		t.Errorf("transaction = %+v, want %+v", got, want)
	}
}

func TestServiceCreateRejectsInvalidInput(t *testing.T) {
	categoryID := uuid.MustParse("0c9d8e7f-6a5b-4c3d-8e1f-0a9b8c7d6e5f")
	date := time.Date(2026, time.November, 12, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		input   transaction.CreateInput
		wantErr error
	}{
		{
			name: "invalid type",
			input: transaction.CreateInput{
				Type:        "refund",
				AccountID:   testAccountID,
				CategoryID:  &categoryID,
				AmountMinor: 100,
				Date:        date,
			},
			wantErr: transaction.ErrInvalidType,
		},
		{
			name: "invalid shape",
			input: transaction.CreateInput{
				Type:        "expense",
				AccountID:   testAccountID,
				AmountMinor: 100,
				Date:        date,
			},
			wantErr: transaction.ErrCategoryRequired,
		},
		{
			name: "invalid amount",
			input: transaction.CreateInput{
				Type:        "expense",
				AccountID:   testAccountID,
				CategoryID:  &categoryID,
				AmountMinor: 0,
				Date:        date,
			},
			wantErr: transaction.ErrAmountOutOfRange,
		},
		{
			name: "missing date",
			input: transaction.CreateInput{
				Type:        "expense",
				AccountID:   testAccountID,
				CategoryID:  &categoryID,
				AmountMinor: 100,
			},
			wantErr: transaction.ErrInvalidDate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creator := creatorFunc(func(
				context.Context,
				transaction.CreateParams,
			) (transaction.Transaction, error) {
				t.Fatal("storage must not be called for invalid input")
				return transaction.Transaction{}, nil
			})

			service := transaction.NewService(
				creator,
				unexpectedReader{t: t},
			)

			got, err := service.Create(
				context.Background(),
				testUserID,
				tt.input,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}

			if got != (transaction.Transaction{}) {
				t.Errorf("expected zero transaction, got %+v", got)
			}
		})
	}
}

func TestServiceCreateStorageError(t *testing.T) {
	storageErr := errors.New("storage unavailable")
	categoryID := uuid.MustParse("0c9d8e7f-6a5b-4c3d-8e1f-0a9b8c7d6e5f")

	creator := creatorFunc(func(
		context.Context,
		transaction.CreateParams,
	) (transaction.Transaction, error) {
		return transaction.Transaction{}, storageErr
	})

	service := transaction.NewService(
		creator,
		unexpectedReader{t: t},
	)

	got, err := service.Create(
		context.Background(),
		testUserID,
		transaction.CreateInput{
			Type:        "expense",
			AccountID:   testAccountID,
			CategoryID:  &categoryID,
			AmountMinor: 100,
			Date:        time.Date(2026, time.November, 12, 0, 0, 0, 0, time.UTC),
		},
	)

	if !errors.Is(err, storageErr) {
		t.Fatalf("error = %v, want wrapped storage error", err)
	}

	if got != (transaction.Transaction{}) {
		t.Errorf("expected zero transaction, got %+v", got)
	}
}
