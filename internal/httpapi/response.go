package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

type errorResponse struct {
	Error errorDetails `json:"error"`
}

type errorDetails struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorMapping struct {
	err     error
	status  int
	code    string
	message string
}

func writeError(
	w http.ResponseWriter,
	status int,
	code string,
	message string,
) {
	writeJSON(w, status, errorResponse{
		Error: errorDetails{
			Code:    code,
			Message: message,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		log.Printf("encode response: %v", err)

		w.Header().Del("ETag")
		status = http.StatusInternalServerError
		data = []byte(
			`{"error":{"code":"internal","message":"Internal server error"}}`,
		)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)

	if _, err := w.Write(data); err != nil {
		log.Printf("write response: %v", err)
	}
}

func writeMappedError(w http.ResponseWriter, mappings []errorMapping, operation string, err error) {
	for _, m := range mappings {
		if errors.Is(err, m.err) {
			writeError(w, m.status, m.code, m.message)
			return
		}
	}
	log.Printf("%s failed: %v", operation, err)
	writeError(w, http.StatusInternalServerError, "internal", "Internal server error")
}
