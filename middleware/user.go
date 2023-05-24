package middleware

import (
	"net/http"
	"olympsis-server/utils"
)

func UserMiddleware() Middleware {
	return func(f http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// user token
			var token string
			token, err := utils.GetTokenFromHeader(r)

			if err != nil {
				http.Error(w, "failed to get token from header", http.StatusUnauthorized)
				return
			}

			sub, _, _, err := utils.ValidateAuthToken(token)

			if err != nil {
				http.Error(w, "invalid auth token", http.StatusUnauthorized)
				return
			} else {
				// add uuid to token
				r.Header.Add("UUID", sub)

				// call next middleware/handler in chain
				f(w, r)
			}
		}
	}
}
