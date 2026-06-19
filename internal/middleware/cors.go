package middleware

import (
	"net/http"
	"strings"
)

// CORS adds Cross-Origin Resource Sharing headers.
func CORS(origins, methods, headers []string) func(http.Handler) http.Handler {
	originStr := "*"
	if len(origins) > 0 && origins[0] != "*" {
		originStr = strings.Join(origins, ", ")
	}

	methodStr := "GET, POST, PUT, DELETE, OPTIONS"
	if len(methods) > 0 {
		methodStr = strings.Join(methods, ", ")
	}

	headerStr := "Accept, Content-Type, Authorization, X-Request-ID"
	if len(headers) > 0 {
		headerStr = strings.Join(headers, ", ")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", originStr)
			w.Header().Set("Access-Control-Allow-Methods", methodStr)
			w.Header().Set("Access-Control-Allow-Headers", headerStr)
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
