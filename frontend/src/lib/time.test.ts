import { describe, it, expect } from "vitest"
import { fmtElapsed, buildForecastTimes } from "./time"

describe("fmtElapsed", () => {
  it("renders sub-hour durations as minutes only", () => {
    expect(fmtElapsed(0)).toBe("0min")
    expect(fmtElapsed(60_000)).toBe("1min")
    expect(fmtElapsed(45 * 60_000)).toBe("45min")
  })

  it("renders multi-hour durations with zero-padded minutes", () => {
    expect(fmtElapsed(60 * 60_000)).toBe("1h 00min")
    expect(fmtElapsed(65 * 60_000)).toBe("1h 05min")
    expect(fmtElapsed(125 * 60_000)).toBe("2h 05min")
  })

  it("rounds milliseconds to the nearest minute", () => {
    expect(fmtElapsed(29_000)).toBe("0min")
    expect(fmtElapsed(31_000)).toBe("1min")
  })
})

describe("buildForecastTimes", () => {
  const t0 = new Date("2025-01-01T10:00:00Z").getTime()
  const t1 = new Date("2025-01-01T11:00:00Z").getTime()

  it("returns zero-filled output when no forecast points are given", () => {
    expect(buildForecastTimes([], [0, 100, 200])).toEqual([0, 0, 0])
  })

  it("clamps points before the first forecast sample to its timestamp", () => {
    const result = buildForecastTimes(
      [
        { distanceM: 100, time: "2025-01-01T10:00:00Z" },
        { distanceM: 200, time: "2025-01-01T11:00:00Z" },
      ],
      [0, 50, 100]
    )
    expect(result[0]).toBe(t0)
    expect(result[1]).toBe(t0)
    expect(result[2]).toBe(t0)
  })

  it("clamps points beyond the last forecast sample to its timestamp", () => {
    const result = buildForecastTimes(
      [
        { distanceM: 0, time: "2025-01-01T10:00:00Z" },
        { distanceM: 100, time: "2025-01-01T11:00:00Z" },
      ],
      [100, 150, 9999]
    )
    expect(result[0]).toBe(t1)
    expect(result[1]).toBe(t1)
    expect(result[2]).toBe(t1)
  })

  it("linearly interpolates timestamps between forecast samples", () => {
    const result = buildForecastTimes(
      [
        { distanceM: 0, time: "2025-01-01T10:00:00Z" },
        { distanceM: 100, time: "2025-01-01T11:00:00Z" },
      ],
      [0, 25, 50, 75, 100]
    )
    expect(result[0]).toBe(t0)
    expect(result[1]).toBe(t0 + 15 * 60_000)
    expect(result[2]).toBe(t0 + 30 * 60_000)
    expect(result[3]).toBe(t0 + 45 * 60_000)
    expect(result[4]).toBe(t1)
  })

  it("matches output length to the track distance count", () => {
    const result = buildForecastTimes(
      [
        { distanceM: 0, time: "2025-01-01T10:00:00Z" },
        { distanceM: 100, time: "2025-01-01T11:00:00Z" },
      ],
      [0, 50, 100, 150]
    )
    expect(result).toHaveLength(4)
  })
})
