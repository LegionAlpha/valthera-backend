package httpapi

import (
	"net/http"

	"github.com/LegionAlpha/valthera-backend/internal/account"
	"github.com/LegionAlpha/valthera-backend/internal/transaction"
	"github.com/google/uuid"
)

func NewHandler(
	accounts *account.Service,
	transactions *transaction.Service,
	devUserID uuid.UUID,
	allowedOrigin string,
) http.Handler {
	mux := http.NewServeMux()

	accountHandler := &accountHandler{
		service:       accounts,
		devUserID:     devUserID,
		allowedOrigin: allowedOrigin,
	}

	transactionHandler := &transactionHandler{
		service:       transactions,
		devUserID:     devUserID,
		allowedOrigin: allowedOrigin,
	}

	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("POST /api/v1/accounts", accountHandler.create)
	mux.HandleFunc(
		"GET /api/v1/accounts/{id}",
		accountHandler.getByID,
	)
	mux.HandleFunc(
		"POST /api/v1/transactions",
		transactionHandler.create,
	)
	mux.HandleFunc(
		"GET /api/v1/transactions/{id}",
		transactionHandler.getByID,
	)

	return mux
}
