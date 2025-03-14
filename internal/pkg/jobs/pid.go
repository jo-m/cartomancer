package jobs

import (
	"crypto/rand"
	"encoding/base64"
)

// This is a random ID unique per OS process.
const randomIdLen = 32

var randomId string

func init() {
	b := make([]byte, randomIdLen)
	_, err := rand.Read(b)
	if err != nil {
		panic("rand.Read() cannot return err")
	}
	randomId = base64.RawStdEncoding.EncodeToString(b)
}
