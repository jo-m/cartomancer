package utl

import (
	"encoding/base64"
	"fmt"

	"jo-m.ch/go/cartomancer/internal/pkg/password"
)

// JWTSecretMinBytes is the minimum decoded length of a JWT secret (512 bits).
const JWTSecretMinBytes = 64

// DecodeJWTSecret decodes a base64-encoded JWT secret and validates
// that it contains at least JWTSecretMinBytes of entropy.
func DecodeJWTSecret(encoded string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("not valid base64: %w", err)
	}
	if len(decoded) < JWTSecretMinBytes {
		return nil, fmt.Errorf("decoded secret is %d bytes, minimum is %d (512 bits)", len(decoded), JWTSecretMinBytes)
	}
	return decoded, nil
}

// GenJWTSecret generates a cryptographically random JWT secret and returns it
// as a base64-encoded string. The decoded value is JWTSecretMinBytes long.
func GenJWTSecret() string {
	return base64.StdEncoding.EncodeToString(password.GenRandBytes(JWTSecretMinBytes))
}
