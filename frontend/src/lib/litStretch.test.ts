import { describe, it, expect } from "vitest"
import { slicePolylineByDistance } from "./litStretch"

// Coordinates are quarter degrees so that midpoint math is exact in binary
// floating point and the expectations can use toEqual.
const pts = [
  { lat: 47.0, lon: 8.0, d: 0 },
  { lat: 47.25, lon: 8.0, d: 100 },
  { lat: 47.25, lon: 8.5, d: 200 },
]

describe("slicePolylineByDistance", () => {
  it("interpolates both boundaries inside segments", () => {
    expect(slicePolylineByDistance(pts, 50, 150)).toEqual([
      [8.0, 47.125],
      [8.0, 47.25],
      [8.25, 47.25],
    ])
  })

  it("clamps a range extending beyond both ends", () => {
    expect(slicePolylineByDistance(pts, -100, 9999)).toEqual([
      [8.0, 47.0],
      [8.0, 47.25],
      [8.5, 47.25],
    ])
  })

  it("keeps interior switchback vertices", () => {
    const back = [
      { lat: 47.0, lon: 8.0, d: 0 },
      { lat: 47.0, lon: 8.5, d: 100 },
      { lat: 47.0, lon: 8.25, d: 200 },
      { lat: 47.0, lon: 9.0, d: 300 },
    ]
    expect(slicePolylineByDistance(back, 50, 250)).toEqual([
      [8.25, 47.0],
      [8.5, 47.0],
      [8.25, 47.0],
      [8.625, 47.0],
    ])
  })

  it("returns an empty array for degenerate ranges and polylines", () => {
    expect(slicePolylineByDistance(pts, 150, 150)).toEqual([])
    expect(slicePolylineByDistance(pts, 300, 400)).toEqual([])
    expect(slicePolylineByDistance(pts, -10, 0)).toEqual([])
    expect(slicePolylineByDistance([], 0, 100)).toEqual([])
    expect(slicePolylineByDistance([pts[0]], 0, 100)).toEqual([])
  })
})
