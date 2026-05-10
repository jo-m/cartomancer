package api

import (
	"net/http"

	"golang.org/x/time/rate"
)

// rateLimit returns middleware enforcing a single global token-bucket rate
// limit of rps requests per second across all requests passing through it.
// Burst capacity equals rps (minimum 1).
//
// If rps is zero or negative, the returned middleware is a no-op pass-through,
// which allows the limit to be disabled via configuration.
//
// On overflow, the middleware responds with HTTP 429.
func rateLimit(rps float64) func(http.Handler) http.Handler {
	if rps <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	burst := max(int(rps), 1)
	lim := rate.NewLimiter(rate.Limit(rps), burst)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !lim.Allow() {
				writeStatusError(w, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
