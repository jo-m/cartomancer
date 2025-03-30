// Package users deals with users.
package users

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// This is not the most secure choice,
// but apparently the only one which is widely supported.
var totpAlg = otp.AlgorithmSHA1

const (
	totpPeriod     = 30
	totpDigits     = otp.DigitsSix
	totpSecretSize = 20
)

var b32NoPadding = base32.StdEncoding.WithPadding(base32.NoPadding)

func generateOTP(userEmail, issuer string) (*otp.Key, []byte, error) {
	opts := totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: userEmail,
		Period:      uint(totpPeriod),
		SecretSize:  uint(totpSecretSize),
		Digits:      totpDigits,
		Algorithm:   totpAlg,
		Rand:        rand.Reader,
	}

	key, err := totp.Generate(opts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate TOTP key: %w", err)
	}

	secretBytes, err := b32NoPadding.DecodeString(key.Secret())
	if err != nil {
		panic("failed to decode secret as b32, this cannot happen")
	}

	return key, secretBytes, nil
}

func validateOTP(userCode string, secret []byte) (bool, error) {
	opts := totp.ValidateOpts{
		Period:    uint(totpPeriod),
		Skew:      0,
		Digits:    totpDigits,
		Algorithm: totpAlg,
	}
	secretStr := b32NoPadding.EncodeToString(secret)
	return totp.ValidateCustom(userCode, secretStr, time.Now(), opts)
}
