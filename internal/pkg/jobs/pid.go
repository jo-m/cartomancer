package jobs

import (
	"crypto/rand"
	"encoding/base64"
)

// This is a random ID unique per OS process.
const randomIDLen = 32

var randomID string

func init() {
	b := make([]byte, randomIDLen)
	_, err := rand.Read(b)
	if err != nil {
		panic("rand.Read() cannot return err")
	}
	randomID = base64.RawStdEncoding.EncodeToString(b)
}
