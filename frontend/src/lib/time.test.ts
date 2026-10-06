import { describe, it, expect } from "vitest"
import {
  fmtElapsed,
  buildForecastTimes,
  snapToQuarter,
  dayOffsetOf,
  earliestForecastStart,
  latestForecastStart,
  clampForecastStart,
} from "./time"

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

describe("snapToQuarter", () => {
  const at = (h: number, m: number, s = 0) =>
    new Date(2026, 0, 15, h, m, s).getTime()

  it("returns timestamps already on a quarter-hour unchanged", () => {
    expect(snapToQuarter(at(8, 0))).toBe(at(8, 0))
    expect(snapToQuarter(at(8, 15))).toBe(at(8, 15))
    expect(snapToQuarter(at(8, 45, 59))).toBe(at(8, 45))
  })

  it("rounds to the nearest quarter-hour", () => {
    expect(snapToQuarter(at(8, 7))).toBe(at(8, 0))
    expect(snapToQuarter(at(8, 8))).toBe(at(8, 15))
    expect(snapToQuarter(at(8, 22))).toBe(at(8, 15))
    expect(snapToQuarter(at(8, 23))).toBe(at(8, 30))
    expect(snapToQuarter(at(14, 53))).toBe(at(15, 0))
  })

  it("rounds across day boundaries", () => {
    expect(snapToQuarter(at(23, 53))).toBe(
      new Date(2026, 0, 16, 0, 0).getTime()
    )
  })
})

describe("dayOffsetOf", () => {
  it("returns 0 for the same calendar day regardless of clock time", () => {
    const now = new Date(2026, 2, 10, 23, 30).getTime()
    expect(dayOffsetOf(new Date(2026, 2, 10, 0, 30).getTime(), now)).toBe(0)
    expect(dayOffsetOf(new Date(2026, 2, 10, 23, 45).getTime(), now)).toBe(0)
  })

  it("counts calendar day differences across midnight", () => {
    const now = new Date(2026, 2, 10, 23, 30).getTime()
    expect(dayOffsetOf(new Date(2026, 2, 11, 0, 15).getTime(), now)).toBe(1)
    expect(dayOffsetOf(new Date(2026, 2, 9, 23, 45).getTime(), now)).toBe(-1)
  })
})

describe("clampForecastStart", () => {
  const now = new Date(2026, 2, 10, 14, 53).getTime()

  it("snaps in-range times to the nearest quarter-hour", () => {
    expect(
      clampForecastStart(new Date(2026, 2, 10, 16, 37).getTime(), now)
    ).toBe(new Date(2026, 2, 10, 16, 30).getTime())
  })

  it("clamps past times to the current quarter-hour", () => {
    expect(earliestForecastStart(now)).toBe(
      new Date(2026, 2, 10, 15, 0).getTime()
    )
    expect(clampForecastStart(new Date(2026, 2, 10, 6, 0).getTime(), now)).toBe(
      new Date(2026, 2, 10, 15, 0).getTime()
    )
  })

  it("clamps times beyond the horizon to now plus the horizon", () => {
    const far = now + 5 * 24 * 60 * 60 * 1000
    expect(latestForecastStart(now)).toBe(
      new Date(2026, 2, 11, 21, 0).getTime()
    )
    expect(clampForecastStart(far, now)).toBe(latestForecastStart(now))
  })

  it("leaves boundary times unchanged", () => {
    expect(clampForecastStart(earliestForecastStart(now), now)).toBe(
      earliestForecastStart(now)
    )
    expect(clampForecastStart(latestForecastStart(now), now)).toBe(
      latestForecastStart(now)
    )
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
