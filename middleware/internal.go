package middleware

import "net/http"

/*
Internal Middleware
  - guards routes that may only be called by trusted internal services (e.g. the bots
    microservice calling back into the server).
  - validates a shared secret on the X-Internal-Secret header.
  - if no secret is configured (local development), the check is skipped.
*/
func InternalMiddleware(secret string) Middleware {
	return func(f http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if secret != "" && r.Header.Get("X-Internal-Secret") != secret {
				http.Error(w, `{ "msg": "unauthorized" }`, http.StatusUnauthorized)
				return
			}
			f(w, r)
		}
	}
}
