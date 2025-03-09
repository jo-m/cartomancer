// Package password offers password and secrets utilities.
package password

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"golang.org/x/crypto/argon2"
)

type argon2idparams struct {
	Time    uint32 `json:"t"`
	Memory  uint32 `json:"m"`
	Threads uint8  `json:"p"`

	keyLenBytes  uint32
	saltLenBytes uint32
}

// See https://pkg.go.dev/golang.org/x/crypto/argon2#IDKey.
var defaultparams = argon2idparams{
	Time:    1,
	Memory:  64 * 1024,
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
func Hash(password string) string {
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
		panic(fmt.Sprintf("failed to marshal: %s", err))
	}

	return string(ret)
}

// Check if a password matches a hash which was previously generated via Hash().
func Check(password, hashed string) bool {
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

	computedKey := argon2.IDKey([]byte(password), []byte(salt), h.Time, h.Memory, h.Threads, uint32(len(key)))
	return subtle.ConstantTimeCompare([]byte(key), []byte(computedKey)) == 1
}
