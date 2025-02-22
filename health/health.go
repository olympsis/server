package health

import (
	"fmt"
	"net/http"
)

func HealthCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Health Check: Service is Healthy!")
	}
}
