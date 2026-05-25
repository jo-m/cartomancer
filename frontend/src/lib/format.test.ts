import { describe, it, expect } from "vitest"
import { formatDistance, formatAscent, formatUVDoseSED } from "./format"

describe("formatDistance", () => {
  it("renders meters as kilometers with one decimal", () => {
    expect(formatDistance(1234)).toBe("1.2 km")
    expect(formatDistance(10000)).toBe("10.0 km")
  })

  it("rounds to the nearest decimal", () => {
    expect(formatDistance(1250)).toBe("1.3 km")
    expect(formatDistance(1249)).toBe("1.2 km")
  })

  it("handles zero", () => {
    expect(formatDistance(0)).toBe("0.0 km")
  })
})

describe("formatAscent", () => {
  it("rounds meters to an integer", () => {
    expect(formatAscent(123.4)).toBe("123 m")
    expect(formatAscent(123.6)).toBe("124 m")
  })

  it("handles zero", () => {
    expect(formatAscent(0)).toBe("0 m")
  })
})

describe("formatUVDoseSED", () => {
  it("renders very small doses as low", () => {
    expect(formatUVDoseSED(0)).toBe("low")
    expect(formatUVDoseSED(0.4)).toBe("low")
  })

  it("renders single-digit doses with one decimal", () => {
    expect(formatUVDoseSED(0.5)).toBe("0.5 SED")
    expect(formatUVDoseSED(1.23)).toBe("1.2 SED")
    expect(formatUVDoseSED(9.94)).toBe("9.9 SED")
  })

  it("renders larger doses as rounded integers", () => {
    expect(formatUVDoseSED(10)).toBe("10 SED")
    expect(formatUVDoseSED(12.4)).toBe("12 SED")
    expect(formatUVDoseSED(12.6)).toBe("13 SED")
  })
})
