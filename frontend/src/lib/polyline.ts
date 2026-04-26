/**
 * Decoder for Google's Encoded Polyline Algorithm Format with the standard
 * precision of 5 decimal places. The encoding stores deltas of integer-scaled
 * lat/lon coordinates as variable-length, ASCII-printable chunks.
 *
 * Mirrors the encoder used on the backend (track.EncodePolyline) so a
 * server-encoded polyline round-trips through the wire and back to lat/lon
 * pairs without precision loss beyond ~1.1m at the equator.
 */

const POLYLINE_PRECISION = 1e5

export interface LatLon {
  lat: number
  lon: number
}

/**
 * Decodes an encoded polyline string into an array of WGS84 lat/lon points.
 * Returns an empty array for empty input.
 *
 * The string is consumed byte-by-byte: each value is read by accumulating
 * 5-bit chunks until a chunk with the high (0x20) bit cleared is seen, and
 * the resulting unsigned integer is then zig-zag decoded back to a signed
 * delta. The first delta is relative to (0, 0); subsequent deltas accumulate.
 */
export function decodePolyline(encoded: string): LatLon[] {
  const out: LatLon[] = []
  if (!encoded) return out

  let index = 0
  let lat = 0
  let lon = 0
  while (index < encoded.length) {
    const [dLat, nextLat] = readValue(encoded, index)
    if (nextLat < 0) break
    lat += dLat
    index = nextLat
    if (index >= encoded.length) break
    const [dLon, nextLon] = readValue(encoded, index)
    if (nextLon < 0) break
    lon += dLon
    index = nextLon
    out.push({ lat: lat / POLYLINE_PRECISION, lon: lon / POLYLINE_PRECISION })
  }
  return out
}

/**
 * Reads a single zig-zag encoded signed integer from the encoded polyline at
 * pos. Returns [value, nextPos] on success or [0, -1] on truncated input.
 */
function readValue(s: string, pos: number): [number, number] {
  let result = 0
  let shift = 0
  let p = pos
  while (true) {
    if (p >= s.length) return [0, -1]
    const b = s.charCodeAt(p) - 63
    p++
    result |= (b & 0x1f) << shift
    shift += 5
    if (b < 0x20) break
    if (shift > 30) return [0, -1]
  }
  // Zig-zag decode.
  const value = result & 1 ? ~(result >>> 1) : result >>> 1
  return [value, p]
}
