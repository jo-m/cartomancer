package utl

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeJWTSecret(t *testing.T) {
	t.Run("valid exact minimum", func(t *testing.T) {
		raw := []byte(strings.Repeat("A", JWTSecretMinBytes))
		encoded := base64.StdEncoding.EncodeToString(raw)
		decoded, err := DecodeJWTSecret(encoded)
		require.NoError(t, err)
		assert.Equal(t, raw, decoded)
	})

	t.Run("valid longer than minimum", func(t *testing.T) {
		raw := []byte(strings.Repeat("B", JWTSecretMinBytes+32))
		encoded := base64.StdEncoding.EncodeToString(raw)
		decoded, err := DecodeJWTSecret(encoded)
		require.NoError(t, err)
		assert.Equal(t, raw, decoded)
	})

	t.Run("too short", func(t *testing.T) {
		raw := []byte(strings.Repeat("C", JWTSecretMinBytes-1))
		encoded := base64.StdEncoding.EncodeToString(raw)
		_, err := DecodeJWTSecret(encoded)
		require.ErrorContains(t, err, "minimum is 64")
	})

	t.Run("not base64", func(t *testing.T) {
		_, err := DecodeJWTSecret("not-valid-base64!!!")
		require.ErrorContains(t, err, "base64")
	})

	t.Run("empty string", func(t *testing.T) {
		_, err := DecodeJWTSecret("")
		require.ErrorContains(t, err, "minimum is 64")
	})
}

func TestGenJWTSecret(t *testing.T) {
	s := GenJWTSecret()
	decoded, err := DecodeJWTSecret(s)
	require.NoError(t, err)
	assert.Len(t, decoded, JWTSecretMinBytes)

	// Two generated secrets must differ.
	s2 := GenJWTSecret()
	assert.NotEqual(t, s, s2)
}
