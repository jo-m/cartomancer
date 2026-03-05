package rest

import (
	"strings"
	"time"
)

// TestEmailJWTSecret is a fixed 48-byte secret for use in tests.
var TestEmailJWTSecret = strings.Repeat("x", emailJWTSecretLenBytes)

// SignEmailTokenForTest wraps signEmailToken for use in external test packages.
func SignEmailTokenForTest(verificationUUID string, expiry time.Duration, secret []byte, issuer string) (string, error) {
	return signEmailToken(verificationUUID, expiry, secret, issuer)
}
