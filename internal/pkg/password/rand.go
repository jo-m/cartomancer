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

// GenRandAlnumString returns a uniformly distributed alphanumeric random string.
// It uses rejection sampling to avoid modulo bias.
func GenRandAlnumString(n uint32) string {
	const maxUnbiased = 248 // largest multiple of 62 fitting in a byte (62*4)
	result := make([]byte, n)
	for i := range result {
		for {
			b := GenRandBytes(1)[0]
			if b < maxUnbiased {
				result[i] = alnum[b%byte(len(alnum))]
				break
			}
		}
	}
	return string(result)
}
