package httpapi

import (
	"net/http"

	"github.com/LegionAlpha/valthera-backend/internal/account"
	"github.com/google/uuid"
)

func NewHandler(
	accounts *account.Service,
	devUserID uuid.UUID,
	allowedOrigin string,
) http.Handler {
	mux := http.NewServeMux()

	accountHandler := &accountHandler{
		service:       accounts,
		devUserID:     devUserID,
		allowedOrigin: allowedOrigin,
	}

	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("POST /api/v1/accounts", accountHandler.create)
	mux.HandleFunc(
		"GET /api/v1/accounts/{id}",
		accountHandler.getByID,
	)

	return mux
}
