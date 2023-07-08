package middleware

import (
	"io"
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

			// Print request headers
			log.Println("\n - Request Headers:")
			r.Header.Write(log.Writer())

			// Read request body
			body, _ := io.ReadAll(r.Body)

			// Print request body
			log.Println("\n - Request Body:")
			b := string(body)
			if b != "" {
				log.Print(b)
			}

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
