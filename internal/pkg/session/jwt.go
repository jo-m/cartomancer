package session

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func init() {
	jwt.TimePrecision = time.Millisecond
}

const jwtClaimSubject = "session"

var (
	jwtAlg = jwt.SigningMethodHS384
)

type jwtClaims struct {
	jwt.RegisteredClaims
}

func claimsForSession(id string, now time.Time, expires time.Duration, issuer string) jwtClaims {
	return jwtClaims{
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expires)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Subject:   jwtClaimSubject,
			ID:        id,
			Issuer:    issuer,
		},
	}
}

func jwtSign(claims jwtClaims, key []byte) (string, error) {
	token := jwt.NewWithClaims(jwtAlg, claims)
	return token.SignedString(key)
}

// ErrInvalidSubject means that a JWT does not contain a subject we consider valid.
var ErrInvalidSubject = errors.New("invalid subject")

func jwtParseAndVerify(token string, now time.Time, key []byte, issuer string) (*jwtClaims, error) {
	// This also validates notbefore/expires.
	parsed, err := jwt.ParseWithClaims(token, &jwtClaims{},
		func(_ *jwt.Token) (any, error) {
			return key, nil
		},
		jwt.WithValidMethods([]string{jwtAlg.Alg()}),
		jwt.WithSubject(jwtClaimSubject),
		jwt.WithTimeFunc(func() time.Time { return now }),
		jwt.WithIssuer(issuer),
	)
	if err != nil {
		return nil, err
	}
	parsedClaims, ok := parsed.Claims.(*jwtClaims)
	if !ok {
		panic("invalid parsed claim type")
	}

	return parsedClaims, nil
}
