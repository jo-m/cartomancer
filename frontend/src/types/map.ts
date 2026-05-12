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
