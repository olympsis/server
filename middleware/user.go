package middleware

import (
	"net/http"
	"olympsis-server/utils"
)

/*
User Middleware
  - used for routes that require the user to be logged in and have an auth token
  - after decoding auth token, add a request header of UUID with user id
*/
func UserMiddleware() Middleware {
	return func(f http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {

			// get auth token from header
			var token string
			token, err := utils.GetTokenFromHeader(r)
			if err != nil {
				http.Error(w, "failed to get token from header", http.StatusUnauthorized)
				return
			}

			/*
				Validating Auth Token
				- if claims are missing throw an error
			*/
			sub, _, _, err := utils.ValidateAuthToken(token)
			if err != nil {
				http.Error(w, "invalid auth token", http.StatusUnauthorized)
				return
			} else {
				// add uuid to header
				r.Header.Add("UUID", sub)

				// call next middleware/handler in chain
				f(w, r)
			}

		}
	}
}
