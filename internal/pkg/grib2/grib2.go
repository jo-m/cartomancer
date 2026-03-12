// Package grib2 provides a minimal parser for the GRIB2 binary meteorological
// data format, targeted at the ICON-CH1-EPS forecast files distributed by
// MeteoSwiss via the Swiss government STAC API.
//
// Only the subset of the format required to decode single-level wind and
// precipitation forecast fields from that model is implemented:
//   - Grid definition template 101 (ICON unstructured triangular grid)
//   - Data representation template 0 (simple packing, up to 32 bits per value)
//   - Product definition templates 1 and 11 (ensemble forecasts, point-in-time
//     and statistical time intervals)
//
// Usage overview:
//
//  1. Call ParseGrid with the horizontal-constants GRIB2 file to obtain the
//     spatial grid (lat/lon per ICON grid point).
//
//  2. Call Parse with a forecast GRIB2 file to obtain decoded Message values.
//
//  3. Use Grid.ValueAt to look up the value nearest to a lat/lon coordinate.
package grib2

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"time"
)

// Param identifies a meteorological variable by its GRIB2 discipline, category,
// and parameter codes (cf. WMO Manual on Codes, GRIB2 Table 4.2 and local
// MeteoSwiss extensions).
type Param [3]uint8

// Discipline returns the GRIB2 discipline.
func (p Param) Discipline() uint8 { return p[0] }

// Category returns the GRIB2 parameter category within the discipline.
func (p Param) Category() uint8 { return p[1] }

// Number returns the GRIB2 parameter number within the category.
func (p Param) Number() uint8 { return p[2] }

// Known Param constants for the ICON-CH1-EPS model.
// These codes are specific to MeteoSwiss and may differ from WMO standard tables.
var (
	// Wind variables at 10 m above ground, instantaneous.
	ParamUWind10m = Param{0, 2, 2}  // U_10M: zonal (eastward) wind component, m/s
	ParamVWind10m = Param{0, 2, 3}  // V_10M: meridional (northward) wind component, m/s
	ParamMaxWind  = Param{0, 2, 22} // VMAX_10M: maximum wind speed over 1 h, m/s

	// Precipitation variables, surface level.
	ParamTotPrec = Param{0, 1, 52} // TOT_PREC: total accumulated precipitation, kg/m²
	ParamRain    = Param{0, 1, 77} // RAIN_GSP: large-scale rain accumulation, kg/m²
	ParamSnow    = Param{0, 1, 56} // SNOW_GSP: large-scale snowfall accumulation, kg/m²
	ParamGraupel = Param{0, 1, 75} // GRAU_GSP: graupel accumulation, kg/m²
)

// Message holds the decoded content of one GRIB2 message.
type Message struct {
	// ReferenceTime is the forecast model initialisation time (from GRIB2 Section 1).
	ReferenceTime time.Time
	// ValidTime is the time at which this forecast field is valid.
	// For instantaneous fields (PDT 1) it equals ReferenceTime plus the lead time.
	// For accumulated or maximum fields (PDT 11) it is the end of the statistical
	// time interval.
	ValidTime time.Time
	// Discipline is the GRIB2 discipline code (0 = meteorology).
	Discipline uint8
	// Category is the GRIB2 parameter category within the discipline.
	Category uint8
	// Parameter is the GRIB2 parameter number (may be a local table code).
	Parameter uint8
	// LevelType is the type of the first fixed surface (e.g. 103 = height above ground).
	LevelType uint8
	// LevelValue is the scaled value of the first fixed surface (e.g. 10 for 10 m).
	LevelValue int32
	// Values contains one float32 per ICON grid point in grid-index order.
	// Indices correspond to the Lats/Lons slices in the companion Grid.
	Values []float32
}

// Param returns the (discipline, category, parameter) identifier of the message.
func (m *Message) Param() Param {
	return Param{m.Discipline, m.Category, m.Parameter}
}

// Parse reads all GRIB2 messages from r and returns them decoded.
// Only simple packing (data representation template 0) is supported; messages
// with other packing methods cause an error.
//
// r should contain the raw bytes of a GRIB2 file.  Multiple concatenated
// messages within a single file are handled correctly.
func Parse(r io.Reader) ([]*Message, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading GRIB2 data: %w", err)
	}
	return parseAll(data)
}

// parseAll decodes all GRIB2 messages in data.
func parseAll(data []byte) ([]*Message, error) {
	var msgs []*Message
	pos := 0
	for pos < len(data) {
		// Scan for the next GRIB magic.
		if pos+4 > len(data) || string(data[pos:pos+4]) != "GRIB" {
			pos++
			continue
		}
		m, advance, err := parseMessage(data[pos:])
		if err != nil {
			return nil, fmt.Errorf("message at offset %d: %w", pos, err)
		}
		if m != nil {
			msgs = append(msgs, m)
		}
		pos += advance
	}
	return msgs, nil
}

// parseMessage decodes a single GRIB2 message starting at data[0].
// It returns the decoded Message (or nil if the message should be skipped),
// the number of bytes consumed, and any error.
func parseMessage(data []byte) (*Message, int, error) {
	if len(data) < 16 {
		return nil, 1, nil
	}
	if string(data[0:4]) != "GRIB" {
		return nil, 1, nil
	}

	// Section 0: Indicator Section (16 bytes).
	discipline := data[6]
	edition := data[7]
	if edition != 2 {
		return nil, 1, fmt.Errorf("unsupported GRIB edition %d (only 2 is supported)", edition)
	}
	totalLen := int(binary.BigEndian.Uint64(data[8:16]))
	if totalLen > len(data) {
		return nil, 1, fmt.Errorf("message claims length %d but only %d bytes available", totalLen, len(data))
	}

	msg := &Message{Discipline: discipline}
	var (
		npts      uint32
		refVal    float32
		binScale  int
		decScale  int
		nbits     uint8
		bitmap    []byte // non-nil when a bitmap is present
		dataBytes []byte
	)

	pos := 16 // after Section 0
	for pos < totalLen-4 {
		if string(data[pos:pos+4]) == "7777" {
			break // Section 8: end sentinel
		}
		if pos+5 > totalLen {
			return nil, totalLen, fmt.Errorf("truncated section at offset %d", pos)
		}
		secLen := int(binary.BigEndian.Uint32(data[pos : pos+4]))
		if secLen < 5 || pos+secLen > totalLen {
			return nil, totalLen, fmt.Errorf("invalid section length %d at offset %d", secLen, pos)
		}
		secNum := data[pos+4]
		sec := data[pos : pos+secLen]

		switch secNum {
		case 1:
			if err := parseSection1(sec, msg); err != nil {
				return nil, totalLen, fmt.Errorf("section 1: %w", err)
			}
		case 3:
			if len(sec) < 10 {
				return nil, totalLen, fmt.Errorf("section 3 too short")
			}
			npts = binary.BigEndian.Uint32(sec[6:10])
			gridTmpl := binary.BigEndian.Uint16(sec[12:14])
			if gridTmpl != 101 {
				// Only the ICON unstructured grid (template 101) is supported.
				return nil, totalLen, fmt.Errorf("unsupported grid template %d (only 101 supported)", gridTmpl)
			}
		case 4:
			if err := parseSection4(sec, msg); err != nil {
				return nil, totalLen, fmt.Errorf("section 4: %w", err)
			}
		case 5:
			var err error
			refVal, binScale, decScale, nbits, err = parseSection5(sec)
			if err != nil {
				return nil, totalLen, fmt.Errorf("section 5: %w", err)
			}
		case 6:
			if len(sec) < 6 {
				return nil, totalLen, fmt.Errorf("section 6 too short")
			}
			// Bitmap indicator 255 means no bitmap; any other value means a
			// bitmap follows in the remaining bytes of this section.
			if sec[5] != 255 {
				bitmap = sec[6:]
			}
		case 7:
			dataBytes = sec[5:]
		}
		pos += secLen
	}

	if npts == 0 {
		return nil, totalLen, fmt.Errorf("grid point count is zero")
	}

	var err error
	msg.Values, err = decodeSimplePacking(dataBytes, int(npts), refVal, binScale, decScale, nbits, bitmap)
	if err != nil {
		return nil, totalLen, fmt.Errorf("decoding data: %w", err)
	}

	return msg, totalLen, nil
}

// parseSection1 decodes the Identification Section into msg.
func parseSection1(sec []byte, msg *Message) error {
	if len(sec) < 21 {
		return fmt.Errorf("too short (%d bytes)", len(sec))
	}
	year := int(binary.BigEndian.Uint16(sec[12:14]))
	month := int(sec[14])
	day := int(sec[15])
	hour := int(sec[16])
	minute := int(sec[17])
	second := int(sec[18])
	msg.ReferenceTime = time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC)
	return nil
}

// parseSection4 decodes the Product Definition Section into msg.
// Product definition templates 1 (ensemble, instantaneous) and 11 (ensemble,
// time interval) are supported.
func parseSection4(sec []byte, msg *Message) error {
	if len(sec) < 23 {
		return fmt.Errorf("too short (%d bytes)", len(sec))
	}
	pdt := binary.BigEndian.Uint16(sec[7:9])
	msg.Category = sec[9]
	msg.Parameter = sec[10]

	timeUnit := sec[17]
	fcstTime := binary.BigEndian.Uint32(sec[18:22])
	msg.LevelType = sec[22]
	msg.LevelValue = int32(binary.BigEndian.Uint32(sec[24:28]))

	switch pdt {
	case 1: // Ensemble, instantaneous.
		msg.ValidTime = msg.ReferenceTime.Add(leadDuration(timeUnit, fcstTime))
	case 11: // Ensemble, statistical time interval.
		// The end of the overall time interval is the valid time.
		// It is stored starting at section octet 38 (sec[37:44]).
		if len(sec) < 44 {
			return fmt.Errorf("PDT 11 too short for end-of-interval (%d bytes)", len(sec))
		}
		endYear := int(binary.BigEndian.Uint16(sec[37:39]))
		endMonth := int(sec[39])
		endDay := int(sec[40])
		endHour := int(sec[41])
		endMin := int(sec[42])
		endSec := int(sec[43])
		msg.ValidTime = time.Date(endYear, time.Month(endMonth), endDay,
			endHour, endMin, endSec, 0, time.UTC)
	default:
		return fmt.Errorf("unsupported product definition template %d (only 1 and 11 supported)", pdt)
	}
	return nil
}

// leadDuration converts a GRIB2 time unit indicator and count to a Duration.
// Only the units actually observed in ICON-CH1 files are handled; others cause
// a zero duration to be returned.
func leadDuration(unit uint8, count uint32) time.Duration {
	switch unit {
	case 0:
		return time.Duration(count) * time.Minute
	case 1:
		return time.Duration(count) * time.Hour
	case 2:
		return time.Duration(count) * 24 * time.Hour
	default:
		return 0
	}
}

// parseSection5 decodes the Data Representation Section.
// Only data representation template 0 (simple packing) is supported.
//
// It returns the reference value, binary scale factor, decimal scale factor,
// and bits-per-value.
func parseSection5(sec []byte) (refVal float32, binScale, decScale int, nbits uint8, err error) {
	if len(sec) < 21 {
		return 0, 0, 0, 0, fmt.Errorf("too short (%d bytes)", len(sec))
	}
	drst := binary.BigEndian.Uint16(sec[9:11])
	if drst != 0 {
		return 0, 0, 0, 0, fmt.Errorf("unsupported data representation template %d (only 0 supported)", drst)
	}
	refVal = math.Float32frombits(binary.BigEndian.Uint32(sec[11:15]))

	// GRIB2 uses a non-standard sign encoding for scale factors:
	// the high bit is the sign (1 = negative) and the remaining 15 bits
	// are the magnitude (i.e. not two's complement).
	rawBin := binary.BigEndian.Uint16(sec[15:17])
	if rawBin&0x8000 != 0 {
		binScale = -int(rawBin & 0x7FFF)
	} else {
		binScale = int(rawBin)
	}
	rawDec := binary.BigEndian.Uint16(sec[17:19])
	if rawDec&0x8000 != 0 {
		decScale = -int(rawDec & 0x7FFF)
	} else {
		decScale = int(rawDec)
	}
	nbits = sec[19]
	return refVal, binScale, decScale, nbits, nil
}

// decodeSimplePacking unpacks npts values from payload using GRIB2 simple packing
// (data representation template 0).
//
// Formula: value[i] = refVal + packed[i] × 2^binScale × 10^(-decScale)
//
// If bitmap is non-nil it must be a bitmask with one bit per grid point
// (MSB first); only points with the corresponding bitmap bit set to 1 have
// a packed value; the remainder receive NaN.
func decodeSimplePacking(payload []byte, npts int, refVal float32, binScale, decScale int, nbits uint8, bitmap []byte) ([]float32, error) {
	vals := make([]float32, npts)

	if nbits == 0 {
		// All values equal the reference value.
		for i := range vals {
			vals[i] = refVal
		}
		return vals, nil
	}

	scale := float32(math.Pow(2, float64(binScale)) / math.Pow(10, float64(decScale)))

	if bitmap != nil {
		// Packed values only exist for grid points where the bitmap is 1.
		packed := 0 // index into the packed value stream
		for i := range npts {
			byteIdx := i / 8
			bitIdx := 7 - (i % 8)
			if byteIdx >= len(bitmap) || (bitmap[byteIdx]>>uint(bitIdx))&1 == 0 {
				vals[i] = float32(math.NaN())
				continue
			}
			p, err := readPackedValue(payload, packed, int(nbits))
			if err != nil {
				return nil, err
			}
			vals[i] = refVal + float32(p)*scale
			packed++
		}
		return vals, nil
	}

	// No bitmap: every grid point has a packed value.
	switch nbits {
	case 8:
		if len(payload) < npts {
			return nil, fmt.Errorf("payload too short for %d 8-bit values", npts)
		}
		for i := range npts {
			vals[i] = refVal + float32(payload[i])*scale
		}
	case 16:
		if len(payload) < npts*2 {
			return nil, fmt.Errorf("payload too short for %d 16-bit values", npts)
		}
		for i := range npts {
			p := binary.BigEndian.Uint16(payload[i*2:])
			vals[i] = refVal + float32(p)*scale
		}
	case 24:
		if len(payload) < npts*3 {
			return nil, fmt.Errorf("payload too short for %d 24-bit values", npts)
		}
		for i := range npts {
			p := uint32(payload[i*3])<<16 | uint32(payload[i*3+1])<<8 | uint32(payload[i*3+2])
			vals[i] = refVal + float32(p)*scale
		}
	case 32:
		if len(payload) < npts*4 {
			return nil, fmt.Errorf("payload too short for %d 32-bit values", npts)
		}
		for i := range npts {
			p := binary.BigEndian.Uint32(payload[i*4:])
			vals[i] = refVal + float32(p)*scale
		}
	default:
		// General bit-stream decoder for non-byte-aligned widths.
		for i := range npts {
			p, err := readPackedValue(payload, i, int(nbits))
			if err != nil {
				return nil, err
			}
			vals[i] = refVal + float32(p)*scale
		}
	}
	return vals, nil
}

// readPackedValue reads the nth nbits-wide unsigned integer from a big-endian
// bit stream stored in payload (MSB of bit stream = MSB of payload[0]).
func readPackedValue(payload []byte, n, nbits int) (uint32, error) {
	bitStart := n * nbits
	var v uint32
	for b := range nbits {
		pos := bitStart + b
		byteIdx := pos / 8
		bitIdx := 7 - (pos % 8)
		if byteIdx >= len(payload) {
			return 0, fmt.Errorf("payload too short at bit %d", pos)
		}
		if (payload[byteIdx]>>uint(bitIdx))&1 == 1 {
			v |= 1 << uint(nbits-1-b)
		}
	}
	return v, nil
}
