package middleware

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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
			log.Println("Request Headers:")
			r.Header.Write(os.Stdout)

			// Read request body
			body, _ := io.ReadAll(r.Body)

			// Print request body
			log.Println("Request Body:")
			log.Println(string(body))

			// start time from now
			start := time.Now()

			defer func() {
				log.Println(r.URL.Path, "completed in", time.Since(start))

				responseHeaders := w.Header()
				fmt.Println("Response Headers:")
				for key, values := range responseHeaders {
					for _, value := range values {
						log.Printf("%s: %s\n", key, value)
					}
				}
			}()

			// call next middleware/handler in chain
			f(w, r)
		}
	}
}
