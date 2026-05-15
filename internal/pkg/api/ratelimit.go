package api

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"jo-m.ch/go/cartomancer/internal/pkg/session"
)

// rateLimitEntry holds a per-IP token-bucket limiter and the time it was last used.
type rateLimitEntry struct {
	lim      *rate.Limiter
	lastSeen time.Time
}

// ipRateLimiter enforces a per-IP token-bucket rate limit with a bounded map.
type ipRateLimiter struct {
	rps            rate.Limit
	burst          int
	trustedProxies int
	ipv6PrefixLen  int
	maxEntries     int
	mu             sync.Mutex
	entries        map[string]*rateLimitEntry
}

// newIPRateLimiter creates an ipRateLimiter and starts a background cleanup goroutine
// that evicts entries idle for more than 5 minutes.
//
// Parameters:
//   - rps: token refill rate (requests per second).
//   - burst: maximum token accumulation; 0 means auto (max(int(rps), 1)).
//   - trustedProxies: number of trusted reverse proxies; 0 uses the direct peer address.
//   - ipv6PrefixLen: IPv6 prefix length (1-128) used to group addresses into one bucket.
//   - maxEntries: cap on the number of distinct IP keys held at once.
func newIPRateLimiter(rps float64, burst, trustedProxies, ipv6PrefixLen, maxEntries int) *ipRateLimiter {
	if burst <= 0 {
		burst = max(int(rps), 1)
	}
	l := &ipRateLimiter{
		rps:            rate.Limit(rps),
		burst:          burst,
		trustedProxies: trustedProxies,
		ipv6PrefixLen:  ipv6PrefixLen,
		maxEntries:     maxEntries,
		entries:        make(map[string]*rateLimitEntry),
	}
	go l.cleanup()
	return l
}

// cleanup runs forever, evicting entries that have not been seen for 5 minutes.
func (l *ipRateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		for key, entry := range l.entries {
			if time.Since(entry.lastSeen) > 5*time.Minute {
				delete(l.entries, key)
			}
		}
		l.mu.Unlock()
	}
}

// allow reports whether the request from the given IP should be allowed through.
// When the entry map is full and a new IP arrives, allow returns true (fail-open)
// so that a flood of unique IPs degrades the limiter rather than blocking everyone.
func (l *ipRateLimiter) allow(r *http.Request) bool {
	key := normalizeIP(clientIP(r, l.trustedProxies), l.ipv6PrefixLen)

	l.mu.Lock()
	entry, ok := l.entries[key]
	if !ok {
		if len(l.entries) >= l.maxEntries {
			l.mu.Unlock()
			return true // fail-open: map is full, do not allocate
		}
		entry = &rateLimitEntry{lim: rate.NewLimiter(l.rps, l.burst)}
		l.entries[key] = entry
	}
	entry.lastSeen = time.Now()
	lim := entry.lim
	l.mu.Unlock()

	return lim.Allow()
}

// clientIP extracts the client IP from r, honouring trustedProxies.
//
// When trustedProxies is 0, the TCP peer address (r.RemoteAddr) is used.
// When trustedProxies is N > 0, the X-Forwarded-For header is split by comma
// and the entry at index len(parts)-1-N is returned as the client IP.
// If the header is absent or has too few parts, r.RemoteAddr is used as fallback.
// Returns nil if the address cannot be parsed.
func clientIP(r *http.Request, trustedProxies int) net.IP {
	if trustedProxies > 0 {
		xff := r.Header.Get("X-Forwarded-For")
		if xff != "" {
			parts := strings.Split(xff, ",")
			idx := len(parts) - 1 - trustedProxies
			if idx >= 0 {
				if ip := net.ParseIP(strings.TrimSpace(parts[idx])); ip != nil {
					return ip
				}
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return net.ParseIP(r.RemoteAddr)
	}
	return net.ParseIP(host)
}

// normalizeIP returns the rate-limit key for ip.
//
// IPv4 addresses are returned as-is (full address).
// IPv6 addresses are masked to the given prefix length so that all addresses
// in the same prefix share one bucket.
// A nil ip returns the string "unknown".
func normalizeIP(ip net.IP, ipv6PrefixLen int) string {
	if ip == nil {
		return "unknown"
	}
	if ip.To4() != nil {
		return ip.String()
	}
	mask := net.CIDRMask(ipv6PrefixLen, 128)
	return ip.Mask(mask).String()
}

// rateLimitByIP returns middleware that enforces a per-IP (or per-IPv6-prefix)
// token-bucket rate limit of rps requests per second. When rps is zero or
// negative the returned middleware is a no-op pass-through. On overflow the
// middleware responds with HTTP 429.
//
// Parameters:
//   - rps: token refill rate; 0 or negative disables the limit.
//   - burst: maximum burst; 0 means auto (max(int(rps), 1)).
//   - trustedProxies: see [clientIP].
//   - ipv6PrefixLen: see [normalizeIP].
//   - maxEntries: cap on tracked IPs; new IPs beyond the cap are allowed through.
func rateLimitByIP(rps float64, burst, trustedProxies, ipv6PrefixLen, maxEntries int) func(http.Handler) http.Handler {
	if rps <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	lim := newIPRateLimiter(rps, burst, trustedProxies, ipv6PrefixLen, maxEntries)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !lim.allow(r) {
				writeStatusError(w, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// uidRateLimitEntry holds a per-user token-bucket limiter and the time it was last used.
type uidRateLimitEntry struct {
	lim      *rate.Limiter
	lastSeen time.Time
}

// uidRateLimiter enforces a per-user-UUID token-bucket rate limit with a bounded map.
type uidRateLimiter struct {
	rps        rate.Limit
	burst      int
	maxEntries int
	mu         sync.Mutex
	entries    map[string]*uidRateLimitEntry
}

// newUIDRateLimiter creates a uidRateLimiter and starts a background cleanup goroutine
// that evicts entries idle for more than 5 minutes.
//
// Parameters:
//   - rps: token refill rate (requests per second).
//   - burst: maximum token accumulation; 0 means auto (max(int(rps), 1)).
//   - maxEntries: cap on the number of distinct user keys held at once.
func newUIDRateLimiter(rps float64, burst, maxEntries int) *uidRateLimiter {
	if burst <= 0 {
		burst = max(int(rps), 1)
	}
	l := &uidRateLimiter{
		rps:        rate.Limit(rps),
		burst:      burst,
		maxEntries: maxEntries,
		entries:    make(map[string]*uidRateLimitEntry),
	}
	go l.cleanup()
	return l
}

// cleanup runs forever, evicting entries that have not been seen for 5 minutes.
func (l *uidRateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		for key, entry := range l.entries {
			if time.Since(entry.lastSeen) > 5*time.Minute {
				delete(l.entries, key)
			}
		}
		l.mu.Unlock()
	}
}

// allow reports whether the request from the given user UUID should be allowed through.
// When the entry map is full and a new UID arrives, allow returns true (fail-open).
func (l *uidRateLimiter) allow(uid string) bool {
	l.mu.Lock()
	entry, ok := l.entries[uid]
	if !ok {
		if len(l.entries) >= l.maxEntries {
			l.mu.Unlock()
			return true // fail-open: map is full, do not allocate
		}
		entry = &uidRateLimitEntry{lim: rate.NewLimiter(l.rps, l.burst)}
		l.entries[uid] = entry
	}
	entry.lastSeen = time.Now()
	lim := entry.lim
	l.mu.Unlock()

	return lim.Allow()
}

// rateLimitByUID returns middleware that enforces a per-authenticated-user
// token-bucket rate limit of rps requests per second. When rps is zero or
// negative the returned middleware is a no-op pass-through. On overflow the
// middleware responds with HTTP 429. If the context carries no authenticated
// user, the request is allowed through (fail-open).
//
// Parameters:
//   - rps: token refill rate; 0 or negative disables the limit.
//   - burst: maximum burst; 0 means auto (max(int(rps), 1)).
//   - maxEntries: cap on tracked user entries; new users beyond the cap are allowed through.
func rateLimitByUID(rps float64, burst, maxEntries int) func(http.Handler) http.Handler {
	if rps <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	lim := newUIDRateLimiter(rps, burst, maxEntries)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := session.GetUser(r.Context())
			if user == nil {
				next.ServeHTTP(w, r)
				return
			}
			if !lim.allow(user.Uuid) {
				writeStatusError(w, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
