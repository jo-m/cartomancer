package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// noopHandler returns 200 OK without writing a body.
func noopHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func newReq(remoteAddr, xff string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.RemoteAddr = remoteAddr
	if xff != "" {
		r.Header.Set("X-Forwarded-For", xff)
	}
	return r
}

// newLimiter creates a limiter with auto burst, no proxies, /64 IPv6, large cap.
func newLimiter(rps float64) func(http.Handler) http.Handler {
	return rateLimitByIP(rps, 0, 0, 64, 100000)
}

func TestRateLimitByIP_DisabledWhenZero(t *testing.T) {
	handler := newLimiter(0)(noopHandler())

	for range 50 {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, newReq("1.2.3.4:1234", ""))
		require.Equal(t, http.StatusOK, rec.Code)
	}
}

func TestRateLimitByIP_DisabledWhenNegative(t *testing.T) {
	handler := newLimiter(-1)(noopHandler())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, newReq("1.2.3.4:1234", ""))
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimitByIP_AllowsBurstThenRejects(t *testing.T) {
	// rps=5 gives burst=5 for a single IP.
	handler := newLimiter(5)(noopHandler())
	req := newReq("1.2.3.4:1234", "")

	for i := range 5 {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equalf(t, http.StatusOK, rec.Code, "request %d should succeed", i)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimitByIP_FractionalRPSHasMinBurstOne(t *testing.T) {
	handler := newLimiter(0.1)(noopHandler())
	req := newReq("1.2.3.4:1234", "")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimitByIP_DifferentIPsAreIndependent(t *testing.T) {
	// rps=2 gives burst=2 per IP; two distinct IPs should each get their own burst.
	handler := newLimiter(2)(noopHandler())

	for _, addr := range []string{"1.1.1.1:1", "2.2.2.2:2"} {
		for i := range 2 {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, newReq(addr, ""))
			require.Equalf(t, http.StatusOK, rec.Code, "IP %s request %d should succeed", addr, i)
		}
	}
}

func TestRateLimitByIP_ExplicitBurst(t *testing.T) {
	// rps=1/60 (~0.0167) with explicit burst=3: first 3 requests pass, fourth is rejected.
	handler := rateLimitByIP(1.0/60, 3, 0, 64, 100000)(noopHandler())
	req := newReq("1.2.3.4:1234", "")

	for i := range 3 {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equalf(t, http.StatusOK, rec.Code, "request %d should succeed", i)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimitByIP_IPv6PrefixGrouping(t *testing.T) {
	// /64 prefix: two addresses in the same /64 share a bucket (burst=1).
	handler := rateLimitByIP(1, 0, 0, 64, 100000)(noopHandler())

	samePrefix1 := "2001:db8::1:1"
	samePrefix2 := "2001:db8::1:2"
	differentPrefix := "2001:db8:1::1" // different /64

	// First address uses the burst.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, newReq("["+samePrefix1+"]:1", ""))
	require.Equal(t, http.StatusOK, rec.Code)

	// Second address in the same /64 hits the shared limit.
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, newReq("["+samePrefix2+"]:1", ""))
	require.Equal(t, http.StatusTooManyRequests, rec.Code)

	// Address in a different /64 gets its own fresh bucket.
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, newReq("["+differentPrefix+"]:1", ""))
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimitByIP_TrustedProxiesXFF(t *testing.T) {
	// trustedProxies=1: real client IP is the second-from-right XFF entry.
	// RemoteAddr is the proxy; the limiter must key on the XFF client IP.
	handler := rateLimitByIP(1, 0, 1, 64, 100000)(noopHandler())

	clientA := "10.0.0.1"
	clientB := "10.0.0.2"
	proxy := "192.168.1.1:9999"

	// Client A exhausts its burst.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, newReq(proxy, clientA+", 192.168.1.1"))
	require.Equal(t, http.StatusOK, rec.Code)

	// Client A is now rate-limited.
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, newReq(proxy, clientA+", 192.168.1.1"))
	require.Equal(t, http.StatusTooManyRequests, rec.Code)

	// Client B has its own bucket and is not limited.
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, newReq(proxy, clientB+", 192.168.1.1"))
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimitByIP_FailsOpenAtCap(t *testing.T) {
	// maxEntries=2: after two distinct IPs fill the map, a third new IP must be
	// allowed through (fail-open) rather than rejected.
	handler := rateLimitByIP(1, 0, 0, 64, 2)(noopHandler())

	// Fill the map by exhausting two IPs' bursts (so they are in the map).
	for _, addr := range []string{"1.1.1.1:1", "2.2.2.2:2"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, newReq(addr, ""))
		require.Equal(t, http.StatusOK, rec.Code)
	}

	// Third new IP: map is full, must fail-open (200).
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, newReq("3.3.3.3:3", ""))
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestClientIP_DirectConnection(t *testing.T) {
	r := newReq("1.2.3.4:5678", "")
	ip := clientIP(r, 0)
	require.Equal(t, "1.2.3.4", ip.String())
}

func TestClientIP_XFFWithTrustedProxies(t *testing.T) {
	cases := []struct {
		xff            string
		trustedProxies int
		want           string
	}{
		// one proxy: client is leftmost, proxy appended on right
		{"10.0.0.1, 192.168.1.1", 1, "10.0.0.1"},
		// two proxies
		{"10.0.0.1, 10.0.0.2, 192.168.1.1", 2, "10.0.0.1"},
		// header too short for trustedProxies count: fall back to RemoteAddr
		{"192.168.1.1", 2, "9.9.9.9"},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("xff=%q,proxies=%d", tc.xff, tc.trustedProxies), func(t *testing.T) {
			r := newReq("9.9.9.9:1", tc.xff)
			ip := clientIP(r, tc.trustedProxies)
			require.NotNil(t, ip)
			require.Equal(t, tc.want, ip.String())
		})
	}
}

func TestNormalizeIP_IPv4(t *testing.T) {
	ip := normalizeIP([]byte{1, 2, 3, 4}, 64)
	require.Equal(t, "1.2.3.4", ip)
}

func TestNormalizeIP_IPv6Prefix(t *testing.T) {
	// 2001:db8::1 masked to /64 should be 2001:db8::
	ip := normalizeIP(
		[]byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x01},
		64,
	)
	require.Equal(t, "2001:db8::", ip)
}

func TestNormalizeIP_Nil(t *testing.T) {
	require.Equal(t, "unknown", normalizeIP(nil, 64))
}
