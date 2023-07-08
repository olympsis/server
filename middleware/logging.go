package middleware

import (
	"log"
	"net/http"
	"time"
)

type Middleware func(http.HandlerFunc) http.HandlerFunc

/*
Tracks on the time it took to complete the request
  - will be used to keep track of performance on server
*/
func Logging() Middleware {
	return func(f http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {

			// start time from now
			start := time.Now()

			defer func() {
				log.Println(r.URL.Path, "completed in", time.Since(start))
			}()

			// call next middleware/handler in chain
			f(w, r)
		}
	}
}
