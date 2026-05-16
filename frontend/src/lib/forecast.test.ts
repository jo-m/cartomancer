import { describe, it, expect } from "vitest"
import { windDirLabel, relWindLabel, headwindComponent } from "./forecast"

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
