package password

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashCheck(t *testing.T) {
	hashed, err := Hash("password123")
	assert.NoError(t, err)
	assert.True(t, Check("password123", hashed))
	assert.False(t, Check("password1234", hashed))
	assert.False(t, Check("password123", "a"+hashed))
}

func TestHashTooLong(t *testing.T) {
	long := string(make([]byte, MaxPasswordLen+1))
	_, err := Hash(long)
	assert.ErrorIs(t, err, ErrTooLong)
}

func TestHashTooShort(t *testing.T) {
	_, err := Hash("short")
	assert.ErrorIs(t, err, ErrTooShort)
}

func BenchmarkHashCheck(b *testing.B) {
	for i := 0; i < b.N; i++ {
		h, _ := Hash("password123")
		Check("password123", h)
	}
}
