package track

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodePolylineGoogleExample(t *testing.T) {
	// The canonical example from Google's Polyline Algorithm documentation.
	pts := Points{
		{Lat: 38.5, Lon: -120.2},
		{Lat: 40.7, Lon: -120.95},
		{Lat: 43.252, Lon: -126.453},
	}
	got := EncodePolyline(pts)
	require.Equal(t, "_p~iF~ps|U_ulLnnqC_mqNvxq`@", got)
}

func TestEncodePolylineEmpty(t *testing.T) {
	require.Equal(t, "", EncodePolyline(nil))
	require.Equal(t, "", EncodePolyline(Points{}))
}

func TestDecodePolylineGoogleExample(t *testing.T) {
	got := DecodePolyline("_p~iF~ps|U_ulLnnqC_mqNvxq`@")
	require.Len(t, got, 3)
	require.InDelta(t, 38.5, got[0].Lat, 1e-5)
	require.InDelta(t, -120.2, got[0].Lon, 1e-5)
	require.InDelta(t, 40.7, got[1].Lat, 1e-5)
	require.InDelta(t, -120.95, got[1].Lon, 1e-5)
	require.InDelta(t, 43.252, got[2].Lat, 1e-5)
	require.InDelta(t, -126.453, got[2].Lon, 1e-5)
}

func TestDecodePolylineEmpty(t *testing.T) {
	require.Nil(t, DecodePolyline(""))
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	pts := Points{
		{Lat: 47.3769, Lon: 8.5417},
		{Lat: 47.3770, Lon: 8.5420},
		{Lat: 47.3771, Lon: 8.5430},
		{Lat: 47.3680, Lon: 8.5500},
	}
	encoded := EncodePolyline(pts)
	require.NotEmpty(t, encoded)
	got := DecodePolyline(encoded)
	require.Len(t, got, len(pts))
	for i, p := range pts {
		require.InDelta(t, p.Lat, got[i].Lat, 1e-5)
		require.InDelta(t, p.Lon, got[i].Lon, 1e-5)
	}
}

func TestEncodeDecodeSinglePoint(t *testing.T) {
	pts := Points{{Lat: 0, Lon: 0}}
	encoded := EncodePolyline(pts)
	got := DecodePolyline(encoded)
	require.Len(t, got, 1)
	require.InDelta(t, 0, got[0].Lat, 1e-9)
	require.InDelta(t, 0, got[0].Lon, 1e-9)
}

func TestEncodeAllPrintableASCII(t *testing.T) {
	pts := Points{
		{Lat: -89.99999, Lon: -179.99999},
		{Lat: 89.99999, Lon: 179.99999},
	}
	encoded := EncodePolyline(pts)
	for _, c := range encoded {
		require.GreaterOrEqual(t, c, rune(63))
		require.LessOrEqual(t, c, rune(126))
	}
}
