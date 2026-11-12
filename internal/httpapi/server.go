package httpapi

import "net/http"

func NewHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", health)

	return mux
}
