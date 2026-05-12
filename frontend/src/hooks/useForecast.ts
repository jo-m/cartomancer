import { useState, useEffect, useCallback, useMemo } from "react"
import { fetchClient } from "../api/client"
import type { ForecastPoint, ForecastUnits, SunEvent } from "../types/forecast"
import { buildForecastTimes } from "../lib/time"

function getStartTime(hoursOffset: number): Date {
  const startDate = new Date()
  startDate.setHours(startDate.getHours() + hoursOffset)
  return startDate
}

const DEFAULT_FORECAST_UNITS: ForecastUnits = {
  temperatureC: "C",
  precipitationRate: "mm/h",
  windSpeedMs: "m/s",
  windDirectionDeg: "deg",
  relativeWindDirectionDeg: "deg",
}

export interface UseForecastResult {
  forecastPoints: ForecastPoint[] | null
  sunEvents: SunEvent[]
  forecastLoading: boolean
  forecastStatus: string | null
  forecastAttribution: { text: string; href: string } | undefined
  forecastUnits: ForecastUnits
  forecastTimes: number[] | undefined
  startHoursOffset: number
  speedKmh: number
  estDurationH: number
  /** Fetches forecast data from the API with the given parameters. */
  fetchForecast: (hoursOffset: number, speed: number) => Promise<void>
  setStartHoursOffset: (h: number) => void
  setSpeedKmh: (s: number) => void
  getStartTime: (hoursOffset: number) => Date
}

/** Manages forecast state and fetching for a single track. */
export function useForecast(
  uuid: string | undefined,
  totalDistanceM: number | undefined,
  trackDistancesM: number[] | undefined,
  onError: (msg: string) => void
): UseForecastResult {
  const [forecastPoints, setForecastPoints] = useState<ForecastPoint[] | null>(
    null
  )
  const [sunEvents, setSunEvents] = useState<SunEvent[]>([])
  const [forecastLoading, setForecastLoading] = useState(false)
  const [forecastStatus, setForecastStatus] = useState<string | null>(null)
  const [forecastAttribution, setForecastAttribution] = useState<
    { text: string; href: string } | undefined
  >(undefined)
  const [forecastUnits, setForecastUnits] = useState<ForecastUnits>(
    DEFAULT_FORECAST_UNITS
  )
  const [startHoursOffset, setStartHoursOffset] = useState(2)
  const [speedKmh, setSpeedKmh] = useState(28)

  const forecastTimes = useMemo(() => {
    if (!forecastPoints || !trackDistancesM || trackDistancesM.length === 0) {
      return undefined
    }
    return buildForecastTimes(forecastPoints, trackDistancesM)
  }, [forecastPoints, trackDistancesM])

  const estDurationH =
    totalDistanceM && speedKmh > 0 ? totalDistanceM / 1000 / speedKmh : 0

  const fetchForecast = useCallback(
    async (hoursOffset: number, speed: number) => {
      if (!uuid) return
      const startDate = getStartTime(hoursOffset)

      setForecastLoading(true)
      setForecastStatus(null)
      try {
        const { data: result, error: apiError } = await fetchClient.GET(
          "/tracks/{uuid}/forecast",
          {
            params: {
              path: { uuid },
              query: {
                startTime: startDate.toISOString(),
                speedKmh: speed,
              },
            },
          }
        )
        if (apiError) {
          throw new Error(
            (apiError as { msg?: string }).msg ?? "Forecast failed"
          )
        }
        if (!result?.points?.length) {
          return
        }
        setForecastStatus(result.forecastStatus ?? null)
        if (result.attribution) {
          setForecastAttribution(result.attribution)
        }
        if (result.units) {
          setForecastUnits(result.units as ForecastUnits)
        }
        setForecastPoints(result.points as ForecastPoint[])
        setSunEvents((result.sunEvents as SunEvent[]) ?? [])
      } catch (err) {
        onError((err as Error).message)
      } finally {
        setForecastLoading(false)
      }
    },
    [uuid, onError]
  )

  useEffect(() => {
    if (uuid) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      fetchForecast(startHoursOffset, speedKmh)
    }
  }, [uuid]) // eslint-disable-line react-hooks/exhaustive-deps

  return {
    forecastPoints,
    sunEvents,
    forecastLoading,
    forecastStatus,
    forecastAttribution,
    forecastUnits,
    forecastTimes,
    startHoursOffset,
    speedKmh,
    estDurationH,
    fetchForecast,
    setStartHoursOffset,
    setSpeedKmh,
    getStartTime,
  }
}
