package users

import (
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOTPGenerate(t *testing.T) {
	key, bytes, err := generateOTP("test@example.org", "ACME")
	assert.NoError(t, err)
	assert.Len(t, key.Secret(), 32)
	assert.Len(t, bytes, 20)
}

func TestOTPValidate(t *testing.T) {
	key, bytes, err := generateOTP("test@example.org", "ACME")
	require.NoError(t, err)

	code, err := totp.GenerateCode(key.Secret(), time.Now())
	require.NoError(t, err)

	ok, err := validateOTP(code, bytes)
	assert.NoError(t, err)
	assert.True(t, ok)

}
