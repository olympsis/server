package middleware

import (
	"net/http"
	"olympsis-server/utils"
)

/*
Club Admin Middleware
  - used for routes that require club admin access
*/
func ClubAdminMiddleware() Middleware {
	return func(f http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {

			uuid := r.Header.Get("UUID")

			// get club token from header
			var token string
			token, err := utils.GetClubTokenFromHeader(r)
			if err != nil {
				http.Error(w, "failed to get club token from header", http.StatusUnauthorized)
				return
			}

			/*
				Validating Club Token
				- if claims are missing throw an error
			*/
			id, rk, err := utils.ValidateClubToken(token, uuid)
			if err != nil {
				http.Error(w, "invalid auth token", http.StatusUnauthorized)
				return
			} else {
				// add club id to header
				r.Header.Add("clubID", id)
				r.Header.Add("clubRole", rk)

				// call next middleware/handler in chain
				f(w, r)
			}

		}
	}
}
