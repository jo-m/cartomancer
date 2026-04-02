// Package password offers password and secrets utilities.
package password

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"golang.org/x/crypto/argon2"
)

// MaxPasswordLen to avoid DOS via Argon2.
const MaxPasswordLen = 512

// ErrTooLong is returned when a password exceeds [MaxPasswordLen] bytes.
var ErrTooLong = errors.New("password exceeds maximum length")

type argon2idparams struct {
	Time    uint32 `json:"t"`
	Memory  uint32 `json:"m"`
	Threads uint8  `json:"p"`

	keyLenBytes  uint32
	saltLenBytes uint32
}

// See https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html.
var defaultparams = argon2idparams{
	Time:    2,
	Memory:  19456,
	Threads: 1,

	keyLenBytes:  32,
	saltLenBytes: 16,
}

type argonHash struct {
	argon2idparams
	SaltB64 string `json:"s"`
	KeyB64  string `json:"k"`
}

// Hash salts and hashes a password.
// The salt and the hash params are serialized in to the returned string.
// Use [Check] to check a generated hash against a password.
// Returns [ErrTooLong] if the password exceeds [MaxPasswordLen] bytes.
func Hash(password string) (string, error) {
	if len(password) > MaxPasswordLen {
		return "", ErrTooLong
	}

	params := defaultparams

	salt := GenRandBytes(params.saltLenBytes)
	key := argon2.IDKey([]byte(password), []byte(salt), params.Time, params.Memory, params.Threads, params.keyLenBytes)

	h := argonHash{
		argon2idparams: params,
		SaltB64:        base64.RawStdEncoding.EncodeToString(salt),
		KeyB64:         base64.RawStdEncoding.EncodeToString(key),
	}

	ret, err := json.Marshal(h)
	if err != nil {
		// This cannot happen.
		panic(fmt.Sprintf("failed to marshal: %s", err))
	}

	return string(ret), nil
}

// Check if a password matches a hash which was previously generated via [Hash].
func Check(password, hashed string) bool {
	if len(password) > MaxPasswordLen {
		return false
	}

	var h argonHash
	err := json.Unmarshal([]byte(hashed), &h)
	if err != nil {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(h.SaltB64)
	if err != nil {
		return false
	}
	key, err := base64.RawStdEncoding.DecodeString(h.KeyB64)
	if err != nil {
		return false
	}

	if len(key) > math.MaxUint32 {
		return false
	}
	// #nosec G115 This is fine.
	keyLen := uint32(len(key))

	computedKey := argon2.IDKey([]byte(password), []byte(salt), h.Time, h.Memory, h.Threads, keyLen)
	return subtle.ConstantTimeCompare([]byte(key), []byte(computedKey)) == 1
}
