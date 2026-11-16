package httpapi

import "net/http"

func ApplyOriginMiddleware(next http.Handler, allowedOrigin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origins := r.Header.Values("Origin")
		if len(origins) != 1 || origins[0] != allowedOrigin {
			writeError(
				w,
				http.StatusForbidden,
				"forbidden_origin",
				"Origin is not allowed",
			)
			return
		}
		next.ServeHTTP(w, r)
	})
}
