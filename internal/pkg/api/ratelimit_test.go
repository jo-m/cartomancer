package api

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/session"
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

func TestRateLimitByIP_FailsClosedAtCap(t *testing.T) {
	// maxEntries=2: after two distinct IPs fill the map, a third new IP must be
	// rejected (fail-closed) so that an attacker cannot bypass the limiter by
	// flooding it with distinct source addresses.
	handler := rateLimitByIP(1, 0, 0, 64, 2)(noopHandler())

	// Fill the map by inserting two IPs (their burst tokens are consumed but the
	// entries remain).
	for _, addr := range []string{"1.1.1.1:1", "2.2.2.2:2"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, newReq(addr, ""))
		require.Equal(t, http.StatusOK, rec.Code)
	}

	// Third new IP: map is full, must fail-closed (429).
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, newReq("3.3.3.3:3", ""))
	require.Equal(t, http.StatusTooManyRequests, rec.Code)

	// An IP already in the map is unaffected by the cap and is still rate-limited
	// per its own bucket (which has already burned its burst, so this 429s too).
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, newReq("1.1.1.1:1", ""))
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
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

// newUIDHandler wraps noopHandler() with a rateLimitByUID middleware and, when uid
// is non-empty, injects a fake db.User into the request context.
func newUIDHandler(rps float64, burst int, uid string) http.Handler {
	mw := rateLimitByUID(rps, burst, 100000)
	inner := mw(noopHandler())
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if uid != "" {
			r = r.WithContext(session.TestWithUser(r.Context(), uid))
		}
		inner.ServeHTTP(w, r)
	})
}

func TestRateLimitByUID_DisabledWhenZero(t *testing.T) {
	handler := newUIDHandler(0, 0, "user-1")

	for range 50 {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
		require.Equal(t, http.StatusOK, rec.Code)
	}
}

func TestRateLimitByUID_DisabledWhenNegative(t *testing.T) {
	handler := newUIDHandler(-1, 0, "user-1")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimitByUID_AllowsBurstThenRejects(t *testing.T) {
	// rps=5 gives burst=5; first 5 requests pass, sixth is rejected.
	handler := newUIDHandler(5, 0, "user-1")

	for i := range 5 {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
		require.Equalf(t, http.StatusOK, rec.Code, "request %d should succeed", i)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimitByUID_ExplicitBurst(t *testing.T) {
	// rps=1/60 (~0.0167) with explicit burst=3: first 3 requests pass, fourth is rejected.
	handler := newUIDHandler(1.0/60, 3, "user-1")

	for i := range 3 {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
		require.Equalf(t, http.StatusOK, rec.Code, "request %d should succeed", i)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimitByUID_DifferentUsersAreIndependent(t *testing.T) {
	// rps=2 gives burst=2 per user; two distinct users each get their own bucket.
	lim := rateLimitByUID(2, 0, 100000)
	inner := lim(noopHandler())

	for _, uid := range []string{"user-a", "user-b"} {
		for i := range 2 {
			rec := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", nil)
			r = r.WithContext(session.TestWithUser(r.Context(), uid))
			inner.ServeHTTP(rec, r)
			require.Equalf(t, http.StatusOK, rec.Code, "user %s request %d should succeed", uid, i)
		}
	}
}

func TestRateLimitByUID_NoUserFailsOpen(t *testing.T) {
	// No user in context: should be allowed through regardless.
	handler := rateLimitByUID(1, 1, 100000)(noopHandler())

	for range 5 {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
		require.Equal(t, http.StatusOK, rec.Code)
	}
}

// captureTraceLogs returns a context whose logger writes JSON records (down to
// trace level) into the returned buffer.
func captureTraceLogs(t *testing.T) (*bytes.Buffer, context.Context) {
	t.Helper()
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: logg.LevelTrace})
	return &buf, logg.WithLogger(t.Context(), slog.New(h))
}

func TestRateLimitByIP_TraceLogsDecisionAndClientIP(t *testing.T) {
	// trustedProxies=1 with a single proxy: the trace log must show the real
	// client IP (10.0.0.1), the raw XFF, the trustedProxies count, and the
	// allowed=true decision for a fresh bucket.
	buf, ctx := captureTraceLogs(t)

	handler := rateLimitByIP(1, 0, 1, 64, 100000)(noopHandler())
	r := newReq("192.168.1.1:9999", "10.0.0.1, 192.168.1.1").WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, r)
	require.Equal(t, http.StatusOK, rec.Code)

	out := buf.String()
	require.Contains(t, out, `"msg":"ip rate limit decision"`)
	require.Contains(t, out, `"remoteAddr":"192.168.1.1:9999"`)
	require.Contains(t, out, `"xff":"10.0.0.1, 192.168.1.1"`)
	require.Contains(t, out, `"trustedProxies":1`)
	require.Contains(t, out, `"clientIP":"10.0.0.1"`)
	require.Contains(t, out, `"key":"10.0.0.1"`)
	require.Contains(t, out, `"allowed":true`)
}

func TestRateLimitByIP_TraceLogsDeniedDecision(t *testing.T) {
	// burst=1: the second request from the same IP must log allowed=false.
	buf, ctx := captureTraceLogs(t)

	handler := rateLimitByIP(1, 0, 0, 64, 100000)(noopHandler())
	req := newReq("1.2.3.4:1234", "").WithContext(ctx)

	handler.ServeHTTP(httptest.NewRecorder(), req)
	buf.Reset()
	handler.ServeHTTP(httptest.NewRecorder(), req)

	require.Contains(t, buf.String(), `"allowed":false`)
}

func TestRateLimitByIP_TraceLogsMapFull(t *testing.T) {
	// maxEntries=1: a second distinct IP must trigger the fail-closed trace log.
	buf, ctx := captureTraceLogs(t)

	handler := rateLimitByIP(1, 0, 0, 64, 1)(noopHandler())
	handler.ServeHTTP(httptest.NewRecorder(), newReq("1.1.1.1:1", "").WithContext(ctx))
	buf.Reset()
	handler.ServeHTTP(httptest.NewRecorder(), newReq("2.2.2.2:2", "").WithContext(ctx))

	require.Contains(t, buf.String(), `"msg":"ip rate limit map full, fail-closed"`)
	require.Contains(t, buf.String(), `"clientIP":"2.2.2.2"`)
}

func TestRateLimitByUID_TraceLogsDecision(t *testing.T) {
	buf, ctx := captureTraceLogs(t)

	mw := rateLimitByUID(5, 0, 100000)
	inner := mw(noopHandler())

	r := httptest.NewRequest(http.MethodPost, "/", nil).WithContext(
		session.TestWithUser(ctx, "user-trace"),
	)
	inner.ServeHTTP(httptest.NewRecorder(), r)

	out := buf.String()
	require.Contains(t, out, `"msg":"uid rate limit decision"`)
	require.Contains(t, out, `"uid":"user-trace"`)
	require.Contains(t, out, `"allowed":true`)
}

func TestRateLimitByUID_TraceLogsFailOpen(t *testing.T) {
	buf, ctx := captureTraceLogs(t)

	handler := rateLimitByUID(1, 1, 100000)(noopHandler())
	r := httptest.NewRequest(http.MethodPost, "/", nil).WithContext(ctx)
	handler.ServeHTTP(httptest.NewRecorder(), r)

	require.Contains(t, buf.String(), `"msg":"uid rate limit fail-open, no user in context"`)
}

func TestRateLimitByUID_FailsClosedAtCap(t *testing.T) {
	// maxEntries=2: after two users fill the map, a third new user must be
	// rejected (fail-closed) so that an attacker cannot bypass the limiter by
	// churning through user keys.
	lim := rateLimitByUID(1, 0, 2)
	inner := lim(noopHandler())

	for _, uid := range []string{"user-a", "user-b"} {
		rec := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r = r.WithContext(session.TestWithUser(r.Context(), uid))
		inner.ServeHTTP(rec, r)
		require.Equal(t, http.StatusOK, rec.Code)
	}

	// Third user: map is full, must fail-closed (429).
	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r = r.WithContext(session.TestWithUser(r.Context(), "user-c"))
	inner.ServeHTTP(rec, r)
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
}
