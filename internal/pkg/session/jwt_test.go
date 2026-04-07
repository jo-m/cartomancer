package session

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/password"
	"jo-m.ch/go/cartomancer/internal/pkg/utl"
)

const issuer = "ACME corp"

func makeTestClaimsAndKey(tb testing.TB) (jwtClaims, []byte) {
	tb.Helper()

	id, err := uuid.NewV7()
	require.NoError(tb, err)
	return claimsForSession(id.String(), time.Now(), time.Hour, issuer), password.GenRandBytes(utl.JWTSecretMinBytes)
}

func TestSimple(t *testing.T) {
	t.Helper()

	claims, key := makeTestClaimsAndKey(t)

	token, err := jwtSign(claims, key)
	assert.NoError(t, err)

	parsed, err := jwtParseAndVerify(token, time.Now(), key, issuer)
	assert.NoError(t, err)

	assert.Equal(t, claims.ID, parsed.ID)
}

// go test -bench=. -run=BenchmarkJWT ./...

func BenchmarkJWTSign(b *testing.B) {
	claims, key := makeTestClaimsAndKey(b)

	for b.Loop() {
		token, err := jwtSign(claims, key)
		require.NoError(b, err)
		require.Greater(b, len(token), 100)
	}
}

func BenchmarkJWTParse(b *testing.B) {
	claims, key := makeTestClaimsAndKey(b)
	token, err := jwtSign(claims, key)
	require.NoError(b, err)
	now := time.Now()

	for b.Loop() {
		parsed, err := jwtParseAndVerify(token, now, key, issuer)
		require.NoError(b, err)
		require.Equal(b, claims.ID, parsed.ID)
	}
}
