package password

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashCheck(t *testing.T) {
	hashed, err := Hash("asdf")
	assert.NoError(t, err)
	assert.True(t, Check("asdf", hashed))
	assert.False(t, Check("asdff", hashed))
	assert.False(t, Check("asdf", "a"+hashed))
}

func TestHashTooLong(t *testing.T) {
	long := string(make([]byte, MaxPasswordLen+1))
	_, err := Hash(long)
	assert.ErrorIs(t, err, ErrTooLong)
}

func BenchmarkHashCheck(b *testing.B) {
	for i := 0; i < b.N; i++ {
		h, _ := Hash("asdf")
		Check("asdf", h)
	}
}
