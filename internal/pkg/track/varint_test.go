package track

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeVarintEmpty(t *testing.T) {
	got, err := EncodeVarint(nil)
	require.NoError(t, err)
	require.Empty(t, got)

	got, err = EncodeVarint(Points{})
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestDecodeVarintEmpty(t *testing.T) {
	got, err := DecodeVarint(nil)
	require.NoError(t, err)
	require.Nil(t, got)

	got, err = DecodeVarint([]byte{})
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestVarintRoundTrip(t *testing.T) {
	pts := Points{
		{Lat: 47.3769, Lon: 8.5417, Elevation: 408.2, Distance: 0},
		{Lat: 47.3770, Lon: 8.5420, Elevation: 408.5, Distance: 32.7},
		{Lat: 47.3771, Lon: 8.5430, Elevation: 410.1, Distance: 110.4},
		{Lat: 47.3680, Lon: 8.5500, Elevation: 415.0, Distance: 1284.6},
	}
	encoded, err := EncodeVarint(pts)
	require.NoError(t, err)
	require.NotEmpty(t, encoded)

	got, err := DecodeVarint(encoded)
	require.NoError(t, err)
	require.Len(t, got, len(pts))
	for i, p := range pts {
		require.InDelta(t, p.Lat, got[i].Lat, 1e-5)
		require.InDelta(t, p.Lon, got[i].Lon, 1e-5)
		require.InDelta(t, p.Elevation, got[i].Elevation, 1e-5)
		// Distance is stored at 1 m resolution.
		require.InDelta(t, p.Distance, got[i].Distance, 0.5)
	}
}

func TestVarintSinglePoint(t *testing.T) {
	pts := Points{{Lat: 0, Lon: 0, Elevation: 0, Distance: 0}}
	encoded, err := EncodeVarint(pts)
	require.NoError(t, err)

	got, err := DecodeVarint(encoded)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.InDelta(t, 0, got[0].Lat, 1e-9)
	require.InDelta(t, 0, got[0].Lon, 1e-9)
	require.InDelta(t, 0, got[0].Elevation, 1e-9)
	require.InDelta(t, 0, got[0].Distance, 1e-9)
}

func TestVarintDistanceRoundTrip(t *testing.T) {
	// Cumulative distances along a 100 km track at 1 m resolution.
	pts := Points{
		{Lat: 47.0, Lon: 8.0, Elevation: 500, Distance: 0},
		{Lat: 47.0, Lon: 8.0001, Elevation: 500, Distance: 7},
		{Lat: 47.0, Lon: 8.001, Elevation: 510, Distance: 81},
		{Lat: 47.01, Lon: 8.01, Elevation: 520, Distance: 1234},
		{Lat: 47.5, Lon: 8.5, Elevation: 800, Distance: 100_000},
	}
	encoded, err := EncodeVarint(pts)
	require.NoError(t, err)

	got, err := DecodeVarint(encoded)
	require.NoError(t, err)
	require.Len(t, got, len(pts))
	for i, p := range pts {
		require.InDeltaf(t, p.Distance, got[i].Distance, 0.5, "distance mismatch at index %d", i)
	}

	// Distances must be monotonic non-decreasing after round-trip.
	for i := 1; i < len(got); i++ {
		require.GreaterOrEqual(t, got[i].Distance, got[i-1].Distance)
	}
}

func TestEncodeVarintRejectsDistanceOverflow(t *testing.T) {
	// Distance precision is 1 m, so any value above ~2.147e9 m overflows int32.
	_, err := EncodeVarint(Points{{Lat: 0, Lon: 0, Elevation: 0, Distance: 3e9}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "distance")
	require.Contains(t, err.Error(), "does not fit")
}

func TestEncodeVarintRejectsDistanceNaN(t *testing.T) {
	_, err := EncodeVarint(Points{{Lat: 0, Lon: 0, Elevation: 0, Distance: math.NaN()}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "distance")
	require.Contains(t, err.Error(), "not finite")
}

func TestVarintNegativeValues(t *testing.T) {
	pts := Points{
		{Lat: -34.6037, Lon: -58.3816, Elevation: 25.0},
		{Lat: -34.6040, Lon: -58.3820, Elevation: -5.5},
	}
	encoded, err := EncodeVarint(pts)
	require.NoError(t, err)

	got, err := DecodeVarint(encoded)
	require.NoError(t, err)
	require.Len(t, got, 2)
	for i, p := range pts {
		require.InDelta(t, p.Lat, got[i].Lat, 1e-5)
		require.InDelta(t, p.Lon, got[i].Lon, 1e-5)
		require.InDelta(t, p.Elevation, got[i].Elevation, 1e-5)
	}
}

func TestEncodeVarintRejectsNaN(t *testing.T) {
	_, err := EncodeVarint(Points{{Lat: math.NaN(), Lon: 0, Elevation: 0}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "latitude")
	require.Contains(t, err.Error(), "not finite")
}

func TestEncodeVarintRejectsInf(t *testing.T) {
	_, err := EncodeVarint(Points{{Lat: 0, Lon: math.Inf(1), Elevation: 0}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "longitude")

	_, err = EncodeVarint(Points{{Lat: 0, Lon: 0, Elevation: math.Inf(-1)}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "elevation")
}

func TestEncodeVarintRejectsOverflow(t *testing.T) {
	// 1e5 * 25000 = 2.5e9, exceeds int32 max (~2.147e9).
	_, err := EncodeVarint(Points{{Lat: 0, Lon: 0, Elevation: 25000}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "elevation")
	require.Contains(t, err.Error(), "does not fit")
}

func TestDecodeVarintTruncated(t *testing.T) {
	pts := Points{
		{Lat: 47.3769, Lon: 8.5417, Elevation: 400},
		{Lat: 47.3770, Lon: 8.5420, Elevation: 401},
	}
	encoded, err := EncodeVarint(pts)
	require.NoError(t, err)

	// Drop the trailing byte to truncate the last varint.
	_, err = DecodeVarint(encoded[:len(encoded)-1])
	require.Error(t, err)
	require.Contains(t, err.Error(), "truncated")
}

func TestDecodeVarintOverflow64(t *testing.T) {
	// Eleven 0xff bytes is a varint that overflows int64.
	bad := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f}
	_, err := DecodeVarint(bad)
	require.Error(t, err)
	require.Contains(t, err.Error(), "overflows 64 bits")
}

func TestDecodeVarintInt32Overflow(t *testing.T) {
	// A varint whose value exceeds int32 max.
	tmp := make([]byte, binary.MaxVarintLen64)
	n := binary.PutVarint(tmp, int64(math.MaxInt32)+1)
	_, err := DecodeVarint(tmp[:n])
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not fit in int32")
}

func TestVarintEndToEndPfanni(t *testing.T) {
	pts := loadGPXPoints(t, "testdata/pfanni highlights.gpx")
	require.NotEmpty(t, pts)

	encoded, err := EncodeVarint(pts)
	require.NoError(t, err)
	require.NotEmpty(t, encoded)

	got, err := DecodeVarint(encoded)
	require.NoError(t, err)
	require.Len(t, got, len(pts))

	for i, p := range pts {
		require.InDeltaf(t, p.Lat, got[i].Lat, 1e-5, "lat mismatch at index %d", i)
		require.InDeltaf(t, p.Lon, got[i].Lon, 1e-5, "lon mismatch at index %d", i)
		require.InDeltaf(t, p.Elevation, got[i].Elevation, 1e-5, "ele mismatch at index %d", i)
		require.InDeltaf(t, p.Distance, got[i].Distance, 0.5, "distance mismatch at index %d", i)
	}

	// Sanity check: with delta + varint encoding the buffer should be much
	// smaller than the naive 24-bytes-per-point representation.
	require.Less(t, len(encoded), len(pts)*24)
}
