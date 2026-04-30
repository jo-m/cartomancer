package track

import (
	"encoding/binary"
	"fmt"
	"math"
)

// varintCoordPrecision is the fixed-point precision used by EncodeVarint and
// DecodeVarint for latitude, longitude, and elevation. Precision 5 (~1.1 m at
// the equator for lat/lon, 1 cm for elevation) matches the polyline encoder.
const varintCoordPrecision = 1e5

// varintDistancePrecision is the fixed-point precision used by EncodeVarint
// and DecodeVarint for cumulative distance. Precision 1 means 1 m resolution,
// which is plenty for chart axes and distance lookups.
const varintDistancePrecision = 1.0

// toFixedPoint converts v to a fixed-point int32 by multiplying with
// precision and rounding to the nearest integer. It returns an error if v is
// not finite or the rounded result does not fit into a signed 32-bit integer.
func toFixedPoint(v, precision float64, name string, idx int) (int32, error) {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("varint encode: point %d %s is not finite: %v", idx, name, v)
	}
	scaled := math.Round(v * precision)
	if scaled > math.MaxInt32 || scaled < math.MinInt32 {
		return 0, fmt.Errorf("varint encode: point %d %s value %v does not fit in int32 fixed-point", idx, name, v)
	}
	return int32(scaled), nil
}

// EncodeVarint encodes pts as a sequence of (latitude, longitude, elevation,
// distance) deltas using the zig-zag varint format from encoding/binary.
// Latitude, longitude, and elevation are quantised to 5-digit fixed point;
// distance uses 1 m resolution. The first point is stored in full as the
// delta against zero. Returns an empty slice for an empty input. Returns an
// error if any value is not finite or does not fit into a signed 32-bit
// fixed-point integer.
func EncodeVarint(pts Points) ([]byte, error) {
	if len(pts) == 0 {
		return []byte{}, nil
	}
	buf := make([]byte, 0, len(pts)*8)
	tmp := make([]byte, binary.MaxVarintLen64)

	var prevLat, prevLon, prevEle, prevDist int32
	for i, p := range pts {
		lat, err := toFixedPoint(p.Lat, varintCoordPrecision, "latitude", i)
		if err != nil {
			return nil, err
		}
		lon, err := toFixedPoint(p.Lon, varintCoordPrecision, "longitude", i)
		if err != nil {
			return nil, err
		}
		ele, err := toFixedPoint(p.Elevation, varintCoordPrecision, "elevation", i)
		if err != nil {
			return nil, err
		}
		dist, err := toFixedPoint(p.Distance, varintDistancePrecision, "distance", i)
		if err != nil {
			return nil, err
		}

		for _, d := range [4]int32{lat - prevLat, lon - prevLon, ele - prevEle, dist - prevDist} {
			n := binary.PutVarint(tmp, int64(d))
			buf = append(buf, tmp[:n]...)
		}
		prevLat, prevLon, prevEle, prevDist = lat, lon, ele, dist
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
		out                             Points
		curLat, curLon, curEle, curDist int32
		off                             int
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
		dDist, n, err := readVarintDelta(b, off, "distance")
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
		newDist, err := addInt32(curDist, dDist, "distance")
		if err != nil {
			return nil, err
		}
		curLat, curLon, curEle, curDist = newLat, newLon, newEle, newDist

		out = append(out, Point{
			Lat:       float64(curLat) / varintCoordPrecision,
			Lon:       float64(curLon) / varintCoordPrecision,
			Elevation: float64(curEle) / varintCoordPrecision,
			Distance:  float64(curDist) / varintDistancePrecision,
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
