package middleware

import "net/http"

/*
Chains up multiple middlewares
*/
func Chain(f http.HandlerFunc, middlewares ...Middleware) http.HandlerFunc {
	for _, m := range middlewares {
		f = m(f)
	}
	return f
}
