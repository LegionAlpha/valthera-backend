package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/LegionAlpha/valthera-backend/internal/account"
	"github.com/google/uuid"
)

type accountHandler struct {
	service       *account.Service
	devUserID     uuid.UUID
	allowedOrigin string
}

type createAccountRequest struct {
	Name                *string `json:"name"`
	InitialBalanceMinor *int64  `json:"initial_balance_minor"`
}

type accountResponse struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	InitialBalanceMinor int64      `json:"initial_balance_minor"`
	ArchivedAt          *time.Time `json:"archived_at"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func (h *accountHandler) create(
	w http.ResponseWriter,
	r *http.Request,
) {
	origins := r.Header.Values("Origin")
	if len(origins) != 1 || origins[0] != h.allowedOrigin {
		writeError(
			w,
			http.StatusForbidden,
			"forbidden_origin",
			"Origin is not allowed",
		)
		return
	}

	if h.devUserID == uuid.Nil {
		writeError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"Authentication required",
		)
		return
	}
	mediaType, _, err := mime.ParseMediaType(
		r.Header.Get("Content-Type"),
	)
	if err != nil || mediaType != "application/json" {
		writeError(
			w,
			http.StatusUnsupportedMediaType,
			"validation_failed",
			"Content-Type must be application/json",
		)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		var sizeErr *http.MaxBytesError

		if errors.As(err, &sizeErr) {
			writeError(
				w,
				http.StatusRequestEntityTooLarge,
				"payload_too_large",
				"Request body exceeds 4096 bytes",
			)
		} else {
			writeError(
				w,
				http.StatusBadRequest,
				"validation_failed",
				"Cannot read request body",
			)
		}
		return
	}

	if !utf8.Valid(body) {
		writeError(
			w,
			http.StatusBadRequest,
			"validation_failed",
			"Request body must be valid UTF-8",
		)
		return
	}
	var request createAccountRequest

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"validation_failed",
			"Invalid request JSON",
		)
		return
	}

	if err := decoder.Decode(new(any)); err != io.EOF {
		writeError(
			w,
			http.StatusBadRequest,
			"validation_failed",
			"Request must contain exactly one JSON object",
		)
		return
	}

	if request.Name == nil || request.InitialBalanceMinor == nil {
		writeError(
			w,
			http.StatusBadRequest,
			"validation_failed",
			"name and initial_balance_minor are required",
		)
		return
	}
	created, err := h.service.Create(
		r.Context(),
		h.devUserID,
		account.CreateInput{
			Name:                *request.Name,
			InitialBalanceMinor: *request.InitialBalanceMinor,
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, account.ErrInvalidName):
			writeError(
				w,
				http.StatusBadRequest,
				"validation_failed",
				"Invalid account name",
			)

		case errors.Is(err, account.ErrInitialBalanceOutOfRange):
			writeError(
				w,
				http.StatusBadRequest,
				"amount_out_of_range",
				"Initial balance is out of range",
			)

		case errors.Is(err, account.ErrAggregateOutOfRange):
			writeError(
				w,
				http.StatusConflict,
				"aggregate_out_of_range",
				"Total balance would exceed the allowed range",
			)

		default:
			log.Printf("create account failed: %v", err)

			writeError(
				w,
				http.StatusInternalServerError,
				"internal",
				"Internal server error",
			)
		}
		return
	}

	writeAccount(w, http.StatusCreated, created)
}

func writeAccount(
	w http.ResponseWriter,
	status int,
	value account.Account,
) {
	var archivedAt *time.Time
	if value.ArchivedAt != nil {
		utc := value.ArchivedAt.UTC()
		archivedAt = &utc
	}

	response := accountResponse{
		ID:                  value.ID.String(),
		Name:                value.Name,
		InitialBalanceMinor: value.InitialBalanceMinor,
		ArchivedAt:          archivedAt,
		CreatedAt:           value.CreatedAt.UTC(),
		UpdatedAt:           value.UpdatedAt.UTC(),
	}

	etag := fmt.Sprintf(
		`"account-v1-%s-%d"`,
		value.ID.String(),
		value.Version,
	)
	w.Header().Set("ETag", etag)

	writeJSON(w, status, response)
}

func (h *accountHandler) getByID(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h.devUserID == uuid.Nil {
		writeError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			"Authentication required",
		)
		return
	}

	accountID, err := uuid.Parse(r.PathValue("id"))
	if err != nil || accountID == uuid.Nil {
		writeError(
			w,
			http.StatusBadRequest,
			"validation_failed",
			"Invalid account ID",
		)
		return
	}

	found, err := h.service.GetByID(
		r.Context(),
		h.devUserID,
		accountID,
	)
	if err != nil {
		switch {
		case errors.Is(err, account.ErrNotFound):
			writeError(
				w,
				http.StatusNotFound,
				"not_found",
				"Account not found",
			)

		case errors.Is(err, account.ErrInvalidAccountID):
			writeError(
				w,
				http.StatusBadRequest,
				"validation_failed",
				"Invalid account ID",
			)

		default:
			log.Printf("get account failed: %v", err)

			writeError(
				w,
				http.StatusInternalServerError,
				"internal",
				"Internal server error",
			)
		}
		return
	}

	writeAccount(w, http.StatusOK, found)
}
