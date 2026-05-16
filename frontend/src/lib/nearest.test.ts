import { describe, it, expect } from "vitest"
import { findNearestIndex } from "./nearest"

const id = (n: number) => n

describe("findNearestIndex", () => {
  it("returns -1 for empty input", () => {
    expect(findNearestIndex([], 5, id)).toBe(-1)
  })

  it("returns 0 for a single-element array", () => {
    expect(findNearestIndex([42], 0, id)).toBe(0)
    expect(findNearestIndex([42], 100, id)).toBe(0)
  })

  it("clamps targets below the first key to index 0", () => {
    expect(findNearestIndex([10, 20, 30], 0, id)).toBe(0)
  })

  it("clamps targets above the last key to the last index", () => {
    expect(findNearestIndex([10, 20, 30], 100, id)).toBe(2)
  })

  it("returns the exact index when the target matches a key", () => {
    expect(findNearestIndex([10, 20, 30], 20, id)).toBe(1)
    expect(findNearestIndex([10, 20, 30], 30, id)).toBe(2)
  })

  it("picks the closer neighbour when the target falls between keys", () => {
    expect(findNearestIndex([0, 10, 20], 12, id)).toBe(1)
    expect(findNearestIndex([0, 10, 20], 18, id)).toBe(2)
  })

  it("prefers the lower index on ties for stability", () => {
    expect(findNearestIndex([0, 10, 20], 5, id)).toBe(0)
    expect(findNearestIndex([0, 10, 20], 15, id)).toBe(1)
  })

  it("works with a custom key accessor", () => {
    const arr = [{ d: 0 }, { d: 100 }, { d: 250 }]
    expect(findNearestIndex(arr, 120, (p) => p.d)).toBe(1)
    expect(findNearestIndex(arr, 200, (p) => p.d)).toBe(2)
  })
})
