package jobs

import (
	"crypto/rand"
	"encoding/binary"
)

// randomID uniquely identifies this process.
// We use a random number instead of os.Getpid() because PIDs can be recycled.
var randomID int64 = genRandInt64()

func genRandInt64() int64 {
	var b [8]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic("rand.Read() cannot return err")
	}
	return int64(binary.BigEndian.Uint64(b[:]))
}
