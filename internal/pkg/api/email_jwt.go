package api

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const emailJWTSubject = "email-verification"

var emailJWTAlg = jwt.SigningMethodHS384

func signEmailToken(verificationUUID string, expiry time.Duration, secret []byte, issuer string) (string, error) {
	now := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		Subject:   emailJWTSubject,
		ID:        verificationUUID,
		Issuer:    issuer,
	}
	token := jwt.NewWithClaims(emailJWTAlg, claims)
	return token.SignedString(secret)
}

func verifyEmailToken(tokenStr string, secret []byte, issuer string) (verificationUUID string, err error) {
	parsed, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{},
		func(_ *jwt.Token) (any, error) {
			return secret, nil
		},
		jwt.WithValidMethods([]string{emailJWTAlg.Alg()}),
		jwt.WithSubject(emailJWTSubject),
		jwt.WithIssuer(issuer),
	)
	if err != nil {
		return "", err
	}
	claims, ok := parsed.Claims.(*jwt.RegisteredClaims)
	if !ok {
		panic("invalid parsed claim type")
	}
	return claims.ID, nil
}
