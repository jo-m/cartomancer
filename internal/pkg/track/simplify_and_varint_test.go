package track

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimplifyAndVarintPreserveCumDist(t *testing.T) {
	pts := loadGPXPoints(t, "testdata/pfanni highlights.gpx")
	require.NotEmpty(t, pts)

	simplified := pts.SimplifyDP(50)
	require.Less(t, len(simplified), len(pts)/10)

	require.Equal(t, pts[0], simplified[0])
	require.Equal(t, pts[len(pts)-1], simplified[len(simplified)-1])

	encoded, err := EncodeVarint(simplified)
	require.NoError(t, err)

	got, err := DecodeVarint(encoded)
	require.NoError(t, err)
	require.Len(t, got, len(simplified))

	fmt.Println(simplified[0], got[0])
	fmt.Println(got[len(got)-1], simplified[len(simplified)-1])

	require.InDeltaf(t, got[0].Distance, simplified[0].Distance, 0.5, "distance mismatch [0]")
	require.InDeltaf(t, got[len(got)-1].Distance, simplified[len(simplified)-1].Distance, 0.5, "distance mismatch [0]")
}
