package track

import (
	"encoding/binary"
	"fmt"
	"math"
)

// varintPrecision is the fixed-point coordinate precision used by EncodeVarint
// and DecodeVarint. Precision 5 (~1.1 m at the equator for lat/lon, 1 cm for
// elevation) matches the polyline encoder.
const varintPrecision = 1e5

// toFixedPoint converts v to a precision-5 fixed-point int32, rounding to the
// nearest integer. It returns an error if v is not finite or the rounded
// result does not fit into a signed 32-bit integer.
func toFixedPoint(v float64, name string, idx int) (int32, error) {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("varint encode: point %d %s is not finite: %v", idx, name, v)
	}
	scaled := math.Round(v * varintPrecision)
	if scaled > math.MaxInt32 || scaled < math.MinInt32 {
		return 0, fmt.Errorf("varint encode: point %d %s value %v does not fit in int32 fixed-point", idx, name, v)
	}
	return int32(scaled), nil
}

// EncodeVarint encodes pts as a sequence of (latitude, longitude, elevation)
// deltas using the zig-zag varint format from encoding/binary. Each value is
// quantized to 5-digit fixed point before differencing; the first point is
// stored in full as the delta against zero. Returns an empty slice for an
// empty input. Returns an error if any value is not finite or does not fit
// into a signed 32-bit fixed-point integer.
func EncodeVarint(pts Points) ([]byte, error) {
	if len(pts) == 0 {
		return []byte{}, nil
	}
	buf := make([]byte, 0, len(pts)*6)
	tmp := make([]byte, binary.MaxVarintLen64)

	var prevLat, prevLon, prevEle int32
	for i, p := range pts {
		lat, err := toFixedPoint(p.Lat, "latitude", i)
		if err != nil {
			return nil, err
		}
		lon, err := toFixedPoint(p.Lon, "longitude", i)
		if err != nil {
			return nil, err
		}
		ele, err := toFixedPoint(p.Elevation, "elevation", i)
		if err != nil {
			return nil, err
		}

		for _, d := range [3]int32{lat - prevLat, lon - prevLon, ele - prevEle} {
			n := binary.PutVarint(tmp, int64(d))
			buf = append(buf, tmp[:n]...)
		}
		prevLat, prevLon, prevEle = lat, lon, ele
	}
	return buf, nil
}

// DecodeVarint decodes a byte slice produced by EncodeVarint back into a
// Points slice. Returns nil for an empty input. Returns an error if the data
// is truncated, contains a varint that overflows 64 bits, or accumulates to a
// fixed-point value outside the signed 32-bit range.
func DecodeVarint(b []byte) (Points, error) {
	if len(b) == 0 {
		return nil, nil
	}
	var (
		out                    Points
		curLat, curLon, curEle int32
		off                    int
	)
	for off < len(b) {
		dLat, n, err := readVarintDelta(b, off, "latitude")
		if err != nil {
			return nil, err
		}
		off += n
		dLon, n, err := readVarintDelta(b, off, "longitude")
		if err != nil {
			return nil, err
		}
		off += n
		dEle, n, err := readVarintDelta(b, off, "elevation")
		if err != nil {
			return nil, err
		}
		off += n

		newLat, err := addInt32(curLat, dLat, "latitude")
		if err != nil {
			return nil, err
		}
		newLon, err := addInt32(curLon, dLon, "longitude")
		if err != nil {
			return nil, err
		}
		newEle, err := addInt32(curEle, dEle, "elevation")
		if err != nil {
			return nil, err
		}
		curLat, curLon, curEle = newLat, newLon, newEle

		out = append(out, Point{
			Lat:       float64(curLat) / varintPrecision,
			Lon:       float64(curLon) / varintPrecision,
			Elevation: float64(curEle) / varintPrecision,
		})
	}
	return out, nil
}

// readVarintDelta reads a single zig-zag varint at b[off:] and validates that
// it fits in an int32. name identifies the field for error messages.
func readVarintDelta(b []byte, off int, name string) (int32, int, error) {
	v, n := binary.Varint(b[off:])
	if n == 0 {
		return 0, 0, fmt.Errorf("varint decode: truncated %s delta at offset %d", name, off)
	}
	if n < 0 {
		return 0, 0, fmt.Errorf("varint decode: %s varint at offset %d overflows 64 bits", name, off)
	}
	if v > math.MaxInt32 || v < math.MinInt32 {
		return 0, 0, fmt.Errorf("varint decode: %s delta %d at offset %d does not fit in int32", name, v, off)
	}
	return int32(v), n, nil
}

// addInt32 returns a + b, returning an error if the result overflows int32.
func addInt32(a, b int32, name string) (int32, error) {
	r := int64(a) + int64(b)
	if r > math.MaxInt32 || r < math.MinInt32 {
		return 0, fmt.Errorf("varint decode: accumulated %s value %d overflows int32", name, r)
	}
	return int32(r), nil
}
