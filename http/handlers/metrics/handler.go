package metrics

import (
	"net/http"

	"github.com/weeb-vip/user/internal/measurements"
)

// Handler tracks metrics about query.
func Handler(metrics measurements.Measurer) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return measurementHandler(metrics, h)
	}
}

func measurementHandler(metrics measurements.Measurer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loggedIn := "true"
		tags := []string{"logged_in:" + loggedIn}

		defer metrics.MeasureExecutionTime("request.time", tags)()
		next.ServeHTTP(w, r)
	})
}
