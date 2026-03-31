package api

import "net/http"

const (
	// csrfHeaderName is the custom header required on all state-changing requests.
	csrfHeaderName = "X-Requested-With"
	// csrfHeaderValue is the expected value of the CSRF header.
	csrfHeaderValue = "cartomancer"
)

// csrfProtect is middleware that rejects non-safe HTTP methods unless the
// request carries the custom X-Requested-With header. Browsers never send
// custom headers on cross-origin requests without a successful CORS preflight,
// so an attacker's page cannot forge the header.
func csrfProtect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			// Safe methods: no check needed.
		default:
			if r.Header.Get(csrfHeaderName) != csrfHeaderValue {
				writeError(w, http.StatusForbidden, "missing or invalid CSRF header")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
