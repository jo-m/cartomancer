/** A single per-point forecast sample along the track. */
export interface ForecastPoint {
  distanceM: number
  time: string
  temperatureC: number | null
  precipitationRate: number | null
  windSpeedMs: number | null
  windDirectionDeg: number | null
  relativeWindDirectionDeg: number | null
}

/** A sun-related event (dawn, sunrise, sunset, dusk) located along the track. */
export interface SunEvent {
  type: "dawn" | "sunrise" | "sunset" | "dusk"
  time: string
  distanceM: number
}

/**
 * Aggregated sun exposure along the track: a dimensionless intensity index in
 * [0, 1] together with the integrated broadband shortwave dose in J/m^2.
 */
export interface SunIntensity {
  index: number
  doseJm2: number
}

/** Unit strings used to label forecast values for display. */
export interface ForecastUnits {
  temperatureC: string
  precipitationRate: string
  windSpeedMs: string
  windDirectionDeg: string
  relativeWindDirectionDeg: string
}
