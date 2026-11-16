package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/LegionAlpha/valthera-backend/internal/idempotency_key"
	"github.com/LegionAlpha/valthera-backend/internal/transaction"
	"github.com/google/uuid"
)

const dateLayout = "2006-01-02"

const createTransactionOperation = "POST /api/v1/transactions"

type transactionHandler struct {
	service        *transaction.Service
	devUserID      uuid.UUID
	allowedOrigin  string
	idempotencyKey *idempotency_key.Service
}

type createTransactionRequest struct {
	Type                 *string `json:"type"`
	AccountID            *string `json:"account_id"`
	DestinationAccountID *string `json:"destination_account_id"`
	CategoryID           *string `json:"category_id"`
	AmountMinor          *int64  `json:"amount_minor"`
	Date                 *string `json:"date"`
	Comment              *string `json:"comment"`
}

type transactionResponse struct {
	ID                   string    `json:"id"`
	Type                 string    `json:"type"`
	AccountID            string    `json:"account_id"`
	DestinationAccountID *string   `json:"destination_account_id"`
	CategoryID           *string   `json:"category_id"`
	AmountMinor          int64     `json:"amount_minor"`
	Date                 string    `json:"date"`
	Comment              string    `json:"comment"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

var transactionValidationMessages = []struct {
	err     error
	message string
}{
	{transaction.ErrInvalidType, "Invalid transaction type"},
	{transaction.ErrInvalidAccountID, "Invalid account ID"},
	{transaction.ErrInvalidCategoryID, "Invalid category ID"},
	{transaction.ErrInvalidDate, "Invalid transaction date"},
	{transaction.ErrInvalidComment, "Invalid transaction comment"},
	{transaction.ErrCategoryRequired, "category_id is required"},
	{transaction.ErrUnexpectedCategory, "Transfer must not have category_id"},
	{transaction.ErrDestinationRequired, "destination_account_id is required"},
	{
		transaction.ErrUnexpectedDestination,
		"Only a transfer can have destination_account_id",
	},
}

func transactionFingerprint(input transaction.CreateInput) []byte {
	return idempotency_key.NewFingerprinter().
		String(input.Type).
		UUID(input.AccountID).
		OptionalUUID(input.DestinationAccountID).
		OptionalUUID(input.CategoryID).
		Int64(input.AmountMinor).
		String(input.Date.Format(dateLayout)).
		String(input.Comment).
		Sum()
}

func parseIdempotencyKey(
	w http.ResponseWriter,
	r *http.Request,
) (string, bool) {
	values := r.Header.Values("Idempotency-Key")

	if len(values) == 0 {
		writeError(
			w,
			http.StatusBadRequest,
			"idempotency_key_missing",
			"Idempotency-Key header is required",
		)
		return "", false
	}

	if len(values) != 1 {
		writeError(
			w,
			http.StatusBadRequest,
			"validation_failed",
			"Invalid Idempotency-Key header",
		)
		return "", false
	}

	key, err := idempotency_key.ValidateKey(values[0])
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"validation_failed",
			"Invalid Idempotency-Key header",
		)
		return "", false
	}

	return key, true
}

func (h *transactionHandler) create(
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

	key, ok := parseIdempotencyKey(w, r)
	if !ok {
		return
	}

	var request createTransactionRequest
	if !decodeJSONBody(w, r, &request) {
		return
	}

	if request.Type == nil ||
		request.AccountID == nil ||
		request.AmountMinor == nil ||
		request.Date == nil {
		writeError(
			w,
			http.StatusBadRequest,
			"validation_failed",
			"type, account_id, amount_minor and date are required",
		)
		return
	}

	input, ok := parseCreateTransactionRequest(w, request)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	scope := idempotency_key.Scope{
		UserID:    h.devUserID,
		Operation: createTransactionOperation,
		Key:       key,
	}

	result, err := h.idempotencyKey.Do(
		ctx,
		scope,
		transactionFingerprint(input),
		func(ctx context.Context) (idempotency_key.Result, error) {
			created, err := h.service.Create(ctx, h.devUserID, input)
			if err != nil {
				return idempotency_key.Result{}, err
			}

			return renderTransaction(http.StatusCreated, created)
		},
	)
	if err != nil {
		writeTransactionError(w, "create transaction", err)
		return
	}

	writeResult(w, result)
}

func (h *transactionHandler) getByID(
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

	transactionID, err := uuid.Parse(r.PathValue("id"))
	if err != nil || transactionID == uuid.Nil {
		writeError(
			w,
			http.StatusBadRequest,
			"validation_failed",
			"Invalid transaction ID",
		)
		return
	}

	found, err := h.service.GetByID(
		r.Context(),
		h.devUserID,
		transactionID,
	)
	if err != nil {
		writeTransactionError(w, "get transaction", err)
		return
	}

	writeTransaction(w, http.StatusOK, found)
}

func parseCreateTransactionRequest(
	w http.ResponseWriter,
	request createTransactionRequest,
) (transaction.CreateInput, bool) {
	accountID, err := uuid.Parse(*request.AccountID)
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"validation_failed",
			"Invalid account ID",
		)
		return transaction.CreateInput{}, false
	}

	destinationAccountID, err := parseOptionalUUID(
		request.DestinationAccountID,
	)
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"validation_failed",
			"Invalid destination account ID",
		)
		return transaction.CreateInput{}, false
	}

	categoryID, err := parseOptionalUUID(request.CategoryID)
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"validation_failed",
			"Invalid category ID",
		)
		return transaction.CreateInput{}, false
	}

	date, err := time.Parse(dateLayout, *request.Date)
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"validation_failed",
			"date must use the YYYY-MM-DD format",
		)
		return transaction.CreateInput{}, false
	}

	var comment string
	if request.Comment != nil {
		comment = *request.Comment
	}

	return transaction.CreateInput{
		Type:                 *request.Type,
		AccountID:            accountID,
		DestinationAccountID: destinationAccountID,
		CategoryID:           categoryID,
		AmountMinor:          *request.AmountMinor,
		Date:                 date,
		Comment:              comment,
	}, true
}

func parseOptionalUUID(raw *string) (*uuid.UUID, error) {
	if raw == nil {
		return nil, nil
	}

	parsed, err := uuid.Parse(*raw)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

func writeTransactionError(
	w http.ResponseWriter,
	operation string,
	err error,
) {
	for _, item := range transactionValidationMessages {
		if errors.Is(err, item.err) {
			writeError(
				w,
				http.StatusBadRequest,
				"validation_failed",
				item.message,
			)
			return
		}
	}

	switch {
	case errors.Is(err, transaction.ErrAmountOutOfRange):
		writeError(
			w,
			http.StatusBadRequest,
			"amount_out_of_range",
			"Amount is out of range",
		)

	case errors.Is(err, transaction.ErrSameAccountTransfer):
		writeError(
			w,
			http.StatusBadRequest,
			"same_account_transfer",
			"Transfer source and destination must differ",
		)

	case errors.Is(err, transaction.ErrInvalidTransactionID):
		writeError(
			w,
			http.StatusBadRequest,
			"validation_failed",
			"Invalid transaction ID",
		)

	case errors.Is(err, transaction.ErrNotFound):
		writeError(
			w,
			http.StatusNotFound,
			"not_found",
			"Transaction not found",
		)

	case errors.Is(err, transaction.ErrAccountNotFound):
		writeError(
			w,
			http.StatusNotFound,
			"not_found",
			"Account not found",
		)

	case errors.Is(err, transaction.ErrAccountArchived):
		writeError(
			w,
			http.StatusConflict,
			"account_archived",
			"Account is archived",
		)

	case errors.Is(err, idempotency_key.ErrKeyReused):
		writeError(
			w,
			http.StatusConflict,
			"idempotency_key_reused",
			"Idempotency-Key was already used with a different request",
		)

	case errors.Is(err, idempotency_key.ErrInProgress):
		writeError(
			w,
			http.StatusConflict,
			"idempotency_in_progress",
			"A request with this Idempotency-Key is still in progress; retry with the same key",
		)

	default:
		log.Printf("%s failed: %v", operation, err)

		writeError(
			w,
			http.StatusInternalServerError,
			"internal",
			"Internal server error",
		)
	}
}

func newTransactionResponse(
	value transaction.Transaction,
) transactionResponse {
	return transactionResponse{
		ID:                   value.ID.String(),
		Type:                 string(value.Type),
		AccountID:            value.AccountID.String(),
		DestinationAccountID: optionalUUIDString(value.DestinationAccountID),
		CategoryID:           optionalUUIDString(value.CategoryID),
		AmountMinor:          value.AmountMinor,
		Date:                 value.Date.Format(dateLayout),
		Comment:              value.Comment,
		CreatedAt:            value.CreatedAt.UTC(),
		UpdatedAt:            value.UpdatedAt.UTC(),
	}
}

func transactionETag(value transaction.Transaction) string {
	return fmt.Sprintf(
		`"transaction-v1-%s-%d"`,
		value.ID.String(),
		value.Version,
	)
}

func writeTransaction(
	w http.ResponseWriter,
	status int,
	value transaction.Transaction,
) {
	w.Header().Set("ETag", transactionETag(value))

	writeJSON(w, status, newTransactionResponse(value))
}

func renderTransaction(
	status int,
	value transaction.Transaction,
) (idempotency_key.Result, error) {
	body, err := json.Marshal(newTransactionResponse(value))
	if err != nil {
		return idempotency_key.Result{}, fmt.Errorf(
			"encode transaction: %w",
			err,
		)
	}

	return idempotency_key.Result{
		ResourceID: value.ID,
		Status:     status,
		Headers: map[string]string{
			"Content-Type": "application/json",
			"ETag":         transactionETag(value),
		},
		Body: body,
	}, nil
}

func writeResult(w http.ResponseWriter, result idempotency_key.Result) {
	for name, value := range result.Headers {
		w.Header().Set(name, value)
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(result.Status)

	if _, err := w.Write(result.Body); err != nil {
		log.Printf("write response: %v", err)
	}
}

func optionalUUIDString(value *uuid.UUID) *string {
	if value == nil {
		return nil
	}

	text := value.String()

	return &text
}
