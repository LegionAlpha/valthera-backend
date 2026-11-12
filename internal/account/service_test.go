package account_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LegionAlpha/valthera-backend/internal/account"
	"github.com/google/uuid"
)

type creatorFunc func(
	context.Context,
	account.CreateParams,
) (account.Account, error)

func (f creatorFunc) Create(
	ctx context.Context,
	params account.CreateParams,
) (account.Account, error) {
	return f(ctx, params)
}

func TestServiceCreate(t *testing.T) {
	userID := uuid.MustParse(
		"d7386164-7cc2-4aa3-a51c-338da77e57a6",
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	now := time.Date(2026, time.November, 12, 10, 0, 0, 0, time.UTC)

	want := account.Account{
		ID:                  uuid.MustParse("8b0e0b8e-e665-42e7-87b9-2f10a228c951"),
		UserID:              userID,
		Name:                "Счёт",
		InitialBalanceMinor: -125050,
		Version:             1,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	calls := 0

	creator := creatorFunc(func(
		gotCtx context.Context,
		params account.CreateParams,
	) (account.Account, error) {
		calls++

		if gotCtx != ctx {
			t.Error("request context was not forwarded")
		}

		wantParams := account.CreateParams{
			UserID:              userID,
			Name:                "Счёт",
			InitialBalanceMinor: -125050,
		}

		if params != wantParams {
			t.Errorf("params = %+v, want %+v", params, wantParams)
		}

		return want, nil
	})

	service := account.NewService(creator)

	got, err := service.Create(ctx, userID, account.CreateInput{
		Name:                "  Сче\u0308т  ",
		InitialBalanceMinor: -125050,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if calls != 1 {
		t.Errorf("storage calls = %d, want 1", calls)
	}

	if got != want {
		t.Errorf("account = %+v, want %+v", got, want)
	}
}

func TestServiceCreateRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		input   account.CreateInput
		wantErr error
	}{
		{
			name: "invalid name",
			input: account.CreateInput{
				Name: "   ",
			},
			wantErr: account.ErrInvalidName,
		},
		{
			name: "invalid balance",
			input: account.CreateInput{
				Name:                "Карта",
				InitialBalanceMinor: 10_000_000_000_001,
			},
			wantErr: account.ErrInitialBalanceOutOfRange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creator := creatorFunc(func(
				context.Context,
				account.CreateParams,
			) (account.Account, error) {
				t.Fatal("storage must not be called for invalid input")
				return account.Account{}, nil
			})

			service := account.NewService(creator)

			got, err := service.Create(
				context.Background(),
				uuid.MustParse("d7386164-7cc2-4aa3-a51c-338da77e57a6"),
				tt.input,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}

			if got != (account.Account{}) {
				t.Errorf("expected zero account, got %+v", got)
			}
		})
	}
}

func TestServiceCreateStorageError(t *testing.T) {
	storageErr := errors.New("storage unavailable")

	creator := creatorFunc(func(
		context.Context,
		account.CreateParams,
	) (account.Account, error) {
		return account.Account{}, storageErr
	})

	service := account.NewService(creator)

	got, err := service.Create(
		context.Background(),
		uuid.MustParse("d7386164-7cc2-4aa3-a51c-338da77e57a6"),
		account.CreateInput{
			Name: "Карта",
		},
	)

	if !errors.Is(err, storageErr) {
		t.Fatalf("error = %v, want wrapped storage error", err)
	}

	if got != (account.Account{}) {
		t.Errorf("expected zero account, got %+v", got)
	}
}
