import { describe, it, expect } from "vitest"
import {
  windDirLabel,
  relWindLabel,
  headwindComponent,
  symmetricHeadwindAxis,
} from "./forecast"

describe("windDirLabel", () => {
  it("maps each octant to its cardinal label", () => {
    expect(windDirLabel(0)).toBe("N")
    expect(windDirLabel(45)).toBe("NE")
    expect(windDirLabel(90)).toBe("E")
    expect(windDirLabel(135)).toBe("SE")
    expect(windDirLabel(180)).toBe("S")
    expect(windDirLabel(225)).toBe("SW")
    expect(windDirLabel(270)).toBe("W")
    expect(windDirLabel(315)).toBe("NW")
  })

  it("wraps 360 degrees back to N", () => {
    expect(windDirLabel(360)).toBe("N")
  })

  it("rounds intermediate angles to the nearest octant", () => {
    expect(windDirLabel(22)).toBe("N")
    expect(windDirLabel(23)).toBe("NE")
    expect(windDirLabel(67)).toBe("NE")
    expect(windDirLabel(68)).toBe("E")
  })
})

describe("relWindLabel", () => {
  it("labels 0 degrees as a pure headwind", () => {
    expect(relWindLabel(0)).toBe("Headwind")
  })

  it("labels 180 degrees as a pure tailwind", () => {
    expect(relWindLabel(180)).toBe("Tailwind")
  })

  it("labels 90 and 270 as crosswinds", () => {
    expect(relWindLabel(90)).toBe("Crosswind R")
    expect(relWindLabel(270)).toBe("Crosswind L")
  })
})

describe("headwindComponent", () => {
  it("returns full wind speed for a pure headwind", () => {
    expect(headwindComponent(10, 0)).toBeCloseTo(10)
  })

  it("returns negative full speed for a pure tailwind", () => {
    expect(headwindComponent(10, 180)).toBeCloseTo(-10)
  })

  it("returns zero for a perpendicular crosswind", () => {
    expect(headwindComponent(10, 90)).toBeCloseTo(0)
    expect(headwindComponent(10, 270)).toBeCloseTo(0)
  })

  it("scales with the cosine of the relative angle", () => {
    expect(headwindComponent(10, 60)).toBeCloseTo(5)
    expect(headwindComponent(10, 120)).toBeCloseTo(-5)
  })
})

describe("symmetricHeadwindAxis", () => {
  it("returns a sensible default for empty input", () => {
    const [[lo, hi], ticks] = symmetricHeadwindAxis([])
    expect(lo).toBe(-hi)
    expect(hi).toBeGreaterThan(0)
    expect(ticks).toContain(0)
  })

  it("is always symmetric around 0", () => {
    const cases = [
      [3, 1, -0.5],
      [-7, -2, -0.1],
      [0.3, -0.2],
      [12, -8],
    ]
    for (const vals of cases) {
      const [[lo, hi], ticks] = symmetricHeadwindAxis(vals)
      expect(lo).toBe(-hi)
      expect(ticks).toContain(0)
      expect(hi).toBeGreaterThanOrEqual(Math.max(...vals.map(Math.abs)))
    }
  })

  it("always includes 0 in the tick list", () => {
    const [, ticks] = symmetricHeadwindAxis([2.3, -1.1, 4.7])
    expect(ticks).toContain(0)
  })

  it("ignores non-finite values and falls back to defaults", () => {
    const [[lo, hi], ticks] = symmetricHeadwindAxis([Infinity, -Infinity, NaN])
    expect(lo).toBe(-hi)
    expect(hi).toBeGreaterThan(0)
    expect(ticks).toContain(0)
  })

  it("snaps to nice step sizes", () => {
    const [, smallTicks] = symmetricHeadwindAxis([1.2, -0.8])
    expect(smallTicks).toEqual([-1.5, -1, -0.5, 0, 0.5, 1, 1.5])
    const [, midTicks] = symmetricHeadwindAxis([5, -3])
    expect(midTicks).toEqual([-6, -4, -2, 0, 2, 4, 6])
  })
})
