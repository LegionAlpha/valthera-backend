package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"unicode/utf8"
)

const maxRequestBodyBytes = 4096

// decodeJSONBody reads exactly one JSON object into dst. On failure it
// writes the error response and returns false.
func decodeJSONBody(
	w http.ResponseWriter,
	r *http.Request,
	dst any,
) bool {
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
		return false
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
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
		return false
	}

	if !utf8.Valid(body) {
		writeError(
			w,
			http.StatusBadRequest,
			"validation_failed",
			"Request body must be valid UTF-8",
		)
		return false
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"validation_failed",
			"Invalid request JSON",
		)
		return false
	}

	if err := decoder.Decode(new(any)); err != io.EOF {
		writeError(
			w,
			http.StatusBadRequest,
			"validation_failed",
			"Request must contain exactly one JSON object",
		)
		return false
	}

	return true
}
