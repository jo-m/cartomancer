package password

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashCheck(t *testing.T) {
	hashed := Hashed("asdf")
	assert.True(t, Check("asdf", hashed))
	assert.False(t, Check("asdff", hashed))
	assert.False(t, Check("asdf", "a"+hashed))
}

// go test -bench=Bench ./...
func BenchmarkPrimeNumbers(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Check("asdf", Hashed("asdf"))
	}
}
