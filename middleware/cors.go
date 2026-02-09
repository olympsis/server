package middleware

import (
	"net/http"
	"os"
)

func CORS() Middleware {
	return func(f http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

			allowedHeaders := "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With"

			// In development mode, allow the userid header for dev auth
			mode := os.Getenv("MODE")
			if mode != "PRODUCTION" {
				allowedHeaders += ", userID"
			}

			w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			// call next middleware/handler in chain
			f(w, r)
		}
	}
}
