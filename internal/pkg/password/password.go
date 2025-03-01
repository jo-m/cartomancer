package password

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"runtime"

	"golang.org/x/crypto/argon2"
)

type argon2idparams struct {
	Time   uint32 `json:"t"`
	Memory uint32 `json:"m"`

	keyLen  uint32
	saltLen uint32
}

var defaultparams argon2idparams = argon2idparams{
	// https://pkg.go.dev/golang.org/x/crypto/argon2#IDKey
	Time:   1,
	Memory: 64 * 1024,

	keyLen:  64,
	saltLen: 32,
}

type argonHash struct {
	argon2idparams
	SaltB64 string `json:"s"`
	KeyB64  string `json:"k"`
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func threads() uint8 {
	cpus := runtime.NumCPU()
	if cpus > 8 {
		return 8
	}
	return uint8(cpus)
}

func Hashed(password string) string {
	params := defaultparams

	salt := GenRandBytes(params.saltLen)
	key := argon2.IDKey([]byte(password), []byte(salt), params.Time, params.Memory, threads(), params.keyLen)

	h := argonHash{
		argon2idparams: params,
		SaltB64:        base64.RawStdEncoding.EncodeToString(salt),
		KeyB64:         base64.RawStdEncoding.EncodeToString(key),
	}

	ret, err := json.Marshal(h)
	if err != nil {
		panic(err)
	}

	return string(ret)
}

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

	computedKey := argon2.IDKey([]byte(password), []byte(salt), h.Time, h.Memory, threads(), uint32(len(key)))
	return subtle.ConstantTimeCompare([]byte(key), []byte(computedKey)) == 1
}
