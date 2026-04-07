package api

import (
	"encoding/base64"
	"strings"
	"time"

	"jo-m.ch/go/cartomancer/internal/pkg/utl"
)

// TestEmailJWTSecret is a fixed base64-encoded secret for use in tests.
var TestEmailJWTSecret = base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", utl.JWTSecretMinBytes)))

// SignEmailTokenForTest wraps signEmailToken for use in external test packages.
func SignEmailTokenForTest(verificationUUID string, expiry time.Duration, secret []byte, issuer string) (string, error) {
	return signEmailToken(verificationUUID, expiry, secret, issuer)
}
