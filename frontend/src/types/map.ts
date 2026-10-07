/** A road closure to display on the map. */
export interface RoadClosure {
  uuid: string
  type: string
  title: string
  startsAt?: string | null
  endsAt?: string | null
  reason?: string | null
  description?: string | null
  geometry: string
  attribution: { text: string; href: string }
}

/** A lit track section, as cumulative distances in meters from the start. */
export interface LitStretch {
  startDistanceM: number
  endDistanceM: number
}
