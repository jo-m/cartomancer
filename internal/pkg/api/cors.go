package api

import "net/http"

// rejectCORS is middleware that explicitly denies all cross-origin requests.
// Browsers send a CORS preflight (OPTIONS with Origin and
// Access-Control-Request-Method headers) before issuing cross-origin requests
// with custom headers. By returning 403 for every preflight, browsers will
// never proceed with the actual request.
//
// This is defense-in-depth: the CSRF header check already blocks cross-origin
// mutations, but this middleware ensures that even if a reverse proxy or future
// middleware accidentally adds permissive Access-Control-Allow-* headers, the
// protection is not silently broken.
func rejectCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions &&
			r.Header.Get("Origin") != "" &&
			r.Header.Get("Access-Control-Request-Method") != "" {
			writeError(w, http.StatusForbidden, "cross-origin requests are not allowed")
			return
		}
		next.ServeHTTP(w, r)
	})
}
