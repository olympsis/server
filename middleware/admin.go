package middleware

import "net/http"

/*
Admin Middleware
  - used for routes that require the user to be an Admin
*/
func AdminMiddleware() Middleware {
	return func(f http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
		}
	}
}
