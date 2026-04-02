package password

import "crypto/rand"

// GeneratedPasswordLen is the length in characters of randomly generated passwords.
const GeneratedPasswordLen = 24

// GenRandBytes returns n cryptographically safe random bytes.
func GenRandBytes(n uint32) []byte {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		panic("rand.Read() cannot return err (see https://github.com/golang/go/issues/66821)")
	}
	return b
}

const alnum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// GenRandAlnumString returns an alphanumeric random string.
func GenRandAlnumString(n uint32) string {
	bytes := GenRandBytes(n)
	for i, b := range bytes {
		bytes[i] = alnum[b%byte(len(alnum))]
	}
	return string(bytes)
}
