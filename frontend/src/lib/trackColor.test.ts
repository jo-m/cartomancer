import { describe, it, expect } from "vitest"
import { trackColorFromUUID } from "./trackColor"

describe("trackColorFromUUID", () => {
  it("returns an HSL string with fixed saturation and lightness", () => {
    const color = trackColorFromUUID("11111111-1111-1111-1111-111111111111")
    expect(color).toMatch(/^hsl\(\d{1,3}, 80%, 45%\)$/)
  })

  it("is deterministic for the same UUID", () => {
    const a = trackColorFromUUID("abcd")
    const b = trackColorFromUUID("abcd")
    expect(a).toBe(b)
  })

  it("produces a hue in [0, 360)", () => {
    for (const uuid of ["a", "b", "c", "long-string-of-text", ""]) {
      const m = trackColorFromUUID(uuid).match(/^hsl\((\d+),/)
      expect(m).not.toBeNull()
      const hue = Number(m![1])
      expect(hue).toBeGreaterThanOrEqual(0)
      expect(hue).toBeLessThan(360)
    }
  })

  it("produces different colors for different UUIDs", () => {
    const a = trackColorFromUUID("track-a")
    const b = trackColorFromUUID("track-b")
    expect(a).not.toBe(b)
  })
})
