package middleware

import (
	"context"
	"fmt"
	"net/http"
	"olympsis-server/utils"

	"firebase.google.com/go/auth"
)

/*
User Middleware
  - used for routes that require the user to be logged in and have an auth token
  - after decoding auth token, add a request header of UUID with user id
*/
func UserMiddleware(auth *auth.Client) Middleware {
	return func(f http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {

			// get auth token from header
			idToken, err := utils.GetTokenFromHeader(r)
			if err != nil {
				fmt.Printf("Failed to get token from header: %s\n", err.Error())
				http.Error(w, `{ "msg": "failed to get token from header" }`, http.StatusUnauthorized)
				return
			}

			/*
				Validating Auth Token
			*/
			token, err := auth.VerifyIDToken(context.TODO(), idToken)

			if err != nil {
				fmt.Printf("Failed to verify token: %s\n", err.Error())
				http.Error(w, `{ "msg": "failed to verify token" }`, http.StatusUnauthorized)
				return
			}
			r.Header.Add("UUID", token.UID)
			f(w, r) // call next middleware/handler in chain
		}
	}
}
