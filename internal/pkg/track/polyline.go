package track

import (
	"math"
	"strings"
)

// polylinePrecision is the coordinate precision used by EncodePolyline and
// DecodePolyline (precision 5, ~1.1m at the equator). This matches Google's
// original "Encoded Polyline Algorithm Format".
const polylinePrecision = 1e5

// EncodePolyline encodes a sequence of points using Google's polyline
// algorithm at precision 5. Only latitude and longitude are encoded;
// elevation and timestamps are ignored. Returns "" for an empty input.
func EncodePolyline(pts Points) string {
	if len(pts) == 0 {
		return ""
	}
	var b strings.Builder
	b.Grow(len(pts) * 8)

	prevLat := int32(0)
	prevLon := int32(0)
	for _, p := range pts {
		lat := int32(math.Round(p.Lat * polylinePrecision))
		lon := int32(math.Round(p.Lon * polylinePrecision))
		appendPolylineValue(&b, lat-prevLat)
		appendPolylineValue(&b, lon-prevLon)
		prevLat = lat
		prevLon = lon
	}
	return b.String()
}

// appendPolylineValue zig-zag encodes a signed int32 delta and writes the
// resulting variable-length quantity into b using the printable-ASCII
// encoding from the Google polyline format.
func appendPolylineValue(b *strings.Builder, v int32) {
	// Zig-zag transform: positive values map to even, negative to odd.
	uv := uint32(v) << 1
	if v < 0 {
		uv = ^uint32(v)<<1 | 1
	}
	for uv >= 0x20 {
		b.WriteByte(byte((0x20 | (uv & 0x1f)) + 63))
		uv >>= 5
	}
	b.WriteByte(byte(uv + 63))
}

// DecodePolyline decodes a string produced by EncodePolyline (precision 5)
// back into a sequence of points. Elevation is left at zero. Returns nil
// for an empty string. A malformed trailing fragment is silently dropped.
func DecodePolyline(s string) Points {
	if s == "" {
		return nil
	}
	var (
		out    Points
		i      int
		curLat int32
		curLon int32
	)
	for i < len(s) {
		dLat, n, ok := readPolylineValue(s, i)
		if !ok {
			break
		}
		i += n
		dLon, n, ok := readPolylineValue(s, i)
		if !ok {
			break
		}
		i += n
		curLat += dLat
		curLon += dLon
		out = append(out, Point{
			Lat: float64(curLat) / polylinePrecision,
			Lon: float64(curLon) / polylinePrecision,
		})
	}
	return out
}

// readPolylineValue reads a single zig-zag encoded value from s starting at
// pos and returns the decoded value, the number of bytes consumed, and a
// success flag. ok is false when the encoded value is truncated.
func readPolylineValue(s string, pos int) (int32, int, bool) {
	var (
		result uint32
		shift  uint
		n      int
	)
	for {
		if pos+n >= len(s) {
			return 0, 0, false
		}
		b := uint32(s[pos+n]) - 63
		n++
		result |= (b & 0x1f) << shift
		shift += 5
		if b < 0x20 {
			break
		}
	}
	if result&1 != 0 {
		return ^int32(result >> 1), n, true
	}
	return int32(result >> 1), n, true
}
