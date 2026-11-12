package account_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/LegionAlpha/valthera-backend/internal/account"
)

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{
			name:  "unchanged",
			input: "Основная карта",
			want:  "Основная карта",
		},
		{
			name:  "trim whitespace",
			input: "\t \u00a0Карта\u00a0 \n",
			want:  "Карта",
		},
		{
			name:  "preserve inner spaces",
			input: "  Моя  карта  ",
			want:  "Моя  карта",
		},
		{
			name:  "normalize NFC",
			input: "Сче\u0308т",
			want:  "Счёт",
		},
		{
			name:  "one code point",
			input: "Я",
			want:  "Я",
		},
		{
			name:  "maximum length",
			input: strings.Repeat("Я", 80),
			want:  strings.Repeat("Я", 80),
		},
		{
			name:  "length after normalization",
			input: " " + strings.Repeat("е\u0308", 80) + " ",
			want:  strings.Repeat("ё", 80),
		},
		{
			name:    "empty",
			input:   "",
			wantErr: account.ErrInvalidName,
		},
		{
			name:    "only whitespace",
			input:   " \t\n\u00a0 ",
			wantErr: account.ErrInvalidName,
		},
		{
			name:    "too long",
			input:   strings.Repeat("Я", 81),
			wantErr: account.ErrInvalidName,
		},
		{
			name:    "invalid UTF-8",
			input:   string([]byte{0xff}),
			wantErr: account.ErrInvalidName,
		},
		{
			name:    "null character",
			input:   "Карта\x00",
			wantErr: account.ErrInvalidName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := account.NormalizeName(tt.input)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}

			if got != tt.want {
				t.Errorf("result = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateInitialBalance(t *testing.T) {
	tests := []struct {
		name    string
		input   int64
		wantErr error
	}{
		{name: "zero", input: 0},
		{name: "positive", input: 125050},
		{name: "negative", input: -125050},
		{name: "minimum", input: -10_000_000_000_000},
		{name: "maximum", input: 10_000_000_000_000},
		{
			name:    "below minimum",
			input:   -10_000_000_000_001,
			wantErr: account.ErrInitialBalanceOutOfRange,
		},
		{
			name:    "above maximum",
			input:   10_000_000_000_001,
			wantErr: account.ErrInitialBalanceOutOfRange,
		},
		{
			name:    "minimum int64",
			input:   -1 << 63,
			wantErr: account.ErrInitialBalanceOutOfRange,
		},
		{
			name:    "maximum int64",
			input:   1<<63 - 1,
			wantErr: account.ErrInitialBalanceOutOfRange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := account.ValidateInitialBalance(tt.input)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
