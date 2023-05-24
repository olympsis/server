package middleware

import "net/http"

/*
Club Admin Middleware
  - used for routes that require club admin access
*/
func ClubAdminMiddleware() Middleware {
	return func(f http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
		}
	}
}
