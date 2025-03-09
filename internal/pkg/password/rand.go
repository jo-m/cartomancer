package password

import "crypto/rand"

// GenRandBytes returns n cryptographically safe random bytes.
func GenRandBytes(n uint32) []byte {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		panic("rand.Read() cannot return err")
	}
	return b
}

// Generated via ('!"' missing, '\' manually removed):
//
//	python3 -c 'print("".join([chr(i) for i in range(35, 127)]))'
const printable = "#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[]^_`abcdefghijklmnopqrstuvwxyz{|}~"

// GenRandPrintableString returns a random string which may contain all printable ASCII chars except '\!"'.
func GenRandPrintableString(n uint32) string {
	bytes := GenRandBytes(n)
	for i, b := range bytes {
		bytes[i] = printable[b%byte(len(printable))]
	}
	return string(bytes)
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
