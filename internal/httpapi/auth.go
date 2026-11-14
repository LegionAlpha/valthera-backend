package httpapi

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/netip"
	"time"

	"github.com/LegionAlpha/valthera-backend/internal/identity"
)

type registrationService interface {
	Register(
		ctx context.Context,
		input identity.RegisterInput,
	) (identity.User, error)
}

type authHandler struct {
	service             registrationService
	registrationEnabled bool
	allowedOrigin       string
	registrationLimiter *registrationLimiter
}

type registerRequest struct {
	Email    *string `json:"email"`
	Password *string `json:"password"`
}

type userResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *authHandler) register(
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

	if !h.registrationEnabled {
		writeError(
			w,
			http.StatusForbidden,
			"registration_disabled",
			"Registration is disabled",
		)
		return
	}

	remote, err := netip.ParseAddrPort(r.RemoteAddr)
	if err != nil {
		log.Println("registration: cannot determine client address")
		writeError(
			w,
			http.StatusInternalServerError,
			"internal",
			"Internal server error",
		)
		return
	}

	if !h.registrationLimiter.allow(remote.Addr().Unmap()) {
		w.Header().Set("Retry-After", "3600")
		writeError(
			w,
			http.StatusTooManyRequests,
			"rate_limited",
			"Too many registration attempts",
		)
		return
	}

	var request registerRequest
	if !decodeJSONBody(w, r, &request) {
		return
	}

	if request.Email == nil || request.Password == nil {
		writeError(
			w,
			http.StatusBadRequest,
			"validation_failed",
			"email and password are required",
		)
		return
	}
	created, err := h.service.Register(
		r.Context(),
		identity.RegisterInput{
			Email:    *request.Email,
			Password: *request.Password,
		},
	)
	if err != nil {
		if errors.Is(err, identity.ErrPasswordHashingBusy) {
			w.Header().Set("Retry-After", "1")
			writeError(
				w,
				http.StatusServiceUnavailable,
				"temporarily_unavailable",
				"Registration is temporarily unavailable",
			)
			return
		}

		writeMappedError(
			w,
			[]errorMapping{
				{
					err:     identity.ErrInvalidEmail,
					status:  http.StatusBadRequest,
					code:    "validation_failed",
					message: "Invalid email",
				},
				{
					err:     identity.ErrInvalidPassword,
					status:  http.StatusBadRequest,
					code:    "validation_failed",
					message: "Invalid password",
				},
				{
					err:     identity.ErrEmailAlreadyExists,
					status:  http.StatusConflict,
					code:    "registration_conflict",
					message: "Email is already registered",
				},
			},
			"register user",
			err,
		)
		return
	}

	if created.Email == nil {
		log.Println("registration returned a user without email")
		writeError(
			w,
			http.StatusInternalServerError,
			"internal",
			"Internal server error",
		)
		return
	}

	writeJSON(w, http.StatusCreated, userResponse{
		ID:        created.ID.String(),
		Email:     *created.Email,
		CreatedAt: created.CreatedAt.UTC(),
	})
}
