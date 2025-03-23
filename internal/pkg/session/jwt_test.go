package session

import (
	"goweb/internal/pkg/password"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestClaimsAndKey(tb testing.TB) (jwtClaims, []byte) {
	id, err := uuid.NewV7()
	require.NoError(tb, err)
	return claimsForSession(id.String(), time.Now(), time.Hour), password.GenRandBytes(jwtSecretLenBytes / 8)
}

func TestSimple(t *testing.T) {
	claims, key := makeTestClaimsAndKey(t)

	token, err := jwtSign(claims, key)
	assert.NoError(t, err)

	parsed, err := jwtParseAndVerify(token, time.Now(), key)
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
		parsed, err := jwtParseAndVerify(token, now, key)
		require.NoError(b, err)
		require.Equal(b, claims.ID, parsed.ID)
	}
}
