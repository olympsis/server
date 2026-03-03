package middleware

import "net/http"

/*
JSONGlobal is a global middleware for use with router.Use().
Sets the default Content-Type to application/json for all responses.
Handlers can override this by calling w.Header().Set("Content-Type", "...") before writing.
*/
func JSONGlobal(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}
