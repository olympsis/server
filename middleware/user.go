package middleware

import (
	"context"
	"net/http"
	"olympsis-server/utils"
	"os"

	"firebase.google.com/go/auth"
)

/*
User Middleware
  - used for routes that require the user to be logged in and have an auth token
  - after decoding auth token, add a request header of UserID with user id
*/
func UserMiddleware(auth *auth.Client) Middleware {
	return func(f http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {

			// Server mode
			if os.Getenv("MODE") == "DEVELOPMENT" {
				if r.Header.Get("UserID") == "" {
					http.Error(w, `{ "msg": "failed to get token from header" }`, http.StatusUnauthorized)
					return
				}
				f(w, r)
				return
			}

			// get auth token from header
			idToken, err := utils.GetTokenFromHeader(r)
			if err != nil {
				http.Error(w, `{ "msg": "failed to get token from header" }`, http.StatusUnauthorized)
				return
			}

			// Validate auth token
			token, err := auth.VerifyIDToken(context.TODO(), idToken)

			if err != nil {
				http.Error(w, `{ "msg": "failed to verify token" }`, http.StatusUnauthorized)
				return
			}
			r.Header.Add("userID", token.UID)
			f(w, r) // call next middleware/handler in chain
		}
	}
}

/*
Optional User Middleware
  - used for routes that work both authenticated and unauthenticated (e.g. public
    listings that change shape when a user is logged in)
  - if a valid token is present, sets the userID header so downstream handlers
    can scope results to the caller; if absent or invalid, the request still
    proceeds without userID set
  - never returns an error response — auth is best-effort here
*/
func OptionalUserMiddleware(auth *auth.Client) Middleware {
	return func(f http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {

			// In DEVELOPMENT mode we trust whatever UserID was supplied by the
			// caller (matching UserMiddleware's dev-mode behavior) but don't
			// require it.
			if os.Getenv("MODE") == "DEVELOPMENT" {
				f(w, r)
				return
			}

			// Best-effort token extraction. Missing token → unauthenticated.
			idToken, err := utils.GetTokenFromHeader(r)
			if err != nil {
				f(w, r)
				return
			}

			// Best-effort token verification. Invalid token → unauthenticated.
			token, err := auth.VerifyIDToken(context.TODO(), idToken)
			if err != nil {
				f(w, r)
				return
			}

			r.Header.Add("userID", token.UID)
			f(w, r)
		}
	}
}
