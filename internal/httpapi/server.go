package httpapi

import (
	"net/http"

	"github.com/LegionAlpha/valthera-backend/internal/account"
	"github.com/LegionAlpha/valthera-backend/internal/idempotency_key"
	"github.com/LegionAlpha/valthera-backend/internal/identity"
	"github.com/LegionAlpha/valthera-backend/internal/transaction"
	"github.com/google/uuid"
)

func NewHandler(
	accounts *account.Service,
	transactions *transaction.Service,
	auth *identity.Service,
	idempotencyKey *idempotency_key.Service,
	devUserID uuid.UUID,
	allowedOrigin string,
	registrationEnabled bool,
) http.Handler {
	mux := http.NewServeMux()

	accountHandler := &accountHandler{
		service:   accounts,
		devUserID: devUserID,
	}

	transactionHandler := &transactionHandler{
		service:        transactions,
		devUserID:      devUserID,
		idempotencyKey: idempotencyKey,
	}

	authHandler := &authHandler{
		service:             auth,
		registrationEnabled: registrationEnabled,
		registrationLimiter: newRegistrationLimiter(),
	}

	mux.HandleFunc("GET /health", health)
	mux.Handle(
		"POST /api/v1/accounts",
		ApplyOriginMiddleware(
			http.HandlerFunc(accountHandler.create),
			allowedOrigin,
		),
	)
	mux.Handle(
		"GET /api/v1/accounts/{id}",
		ApplyOriginMiddleware(
			http.HandlerFunc(accountHandler.getByID),
			allowedOrigin,
		),
	)
	mux.Handle(
		"POST /api/v1/transactions",
		ApplyOriginMiddleware(
			http.HandlerFunc(transactionHandler.create),
			allowedOrigin,
		),
	)
	mux.Handle(
		"GET /api/v1/transactions/{id}",
		ApplyOriginMiddleware(
			http.HandlerFunc(transactionHandler.getByID),
			allowedOrigin,
		),
	)
	mux.Handle(
		"POST /api/v1/auth/register",
		ApplyOriginMiddleware(
			http.HandlerFunc(authHandler.register),
			allowedOrigin,
		),
	)

	return mux
}
