package roadclosures

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClosureTypeString(t *testing.T) {
	cases := []struct {
		ct   ClosureType
		want string
	}{
		{ClosedWay, "closed_way"},
		{Detour, "detour"},
		{Obstruction, "obstruction"},
		{ClosureType(0), "unknown"},
		{ClosureType(99), "unknown"},
	}
	for _, tc := range cases {
		require.Equal(t, tc.want, tc.ct.String())
	}
}
