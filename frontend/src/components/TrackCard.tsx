import { Link } from "react-router-dom"
import SvgPreview from "./SvgPreview"
import StarIcon from "../assets/StarIcon"
import SvgIcon from "../assets/SvgIcon"
import distanceSvg from "../assets/distance.svg?raw"
import elevationSvg from "../assets/elevation.svg?raw"
import temperatureSvg from "../assets/temperature.svg?raw"
import rainSvg from "../assets/rain.svg?raw"
import cardCornerSvg from "../assets/card-corner.svg?raw"
import MiniWindRose from "./MiniWindRose"

function formatDistance(m: number): string {
  return `${(m / 1000).toFixed(1)} km`
}

function formatAscent(m: number): string {
  return `${Math.round(m)} m`
}

interface TrackForecast {
  forecastReferenceTime: string
  startTime: string
  avgTemperatureC?: number | null
  totalPrecipitationMm?: number | null
  windHeadMs?: number
  windRightMs?: number
  windTailMs?: number
  windLeftMs?: number
}

interface TrackData {
  uuid: string
  name: string
  totalDistanceM: number
  totalAscentM: number
  starred?: boolean
  isOwner?: boolean
  user: { uuid: string; name: string }
  forecast?: TrackForecast | null
}

export interface TrackCardProps {
  track: TrackData
  index: number
  isSelected: boolean
  selectionActive: boolean
  canSelect: boolean
  showStar: boolean
  onToggleStar: (e: React.MouseEvent, uuid: string, starred: boolean) => void
  onSelect: (e: React.MouseEvent, uuid: string, index: number) => void
}

/** Renders a single track card with preview, stats, forecast, and selection/star controls. */
export default function TrackCard({
  track,
  index,
  isSelected,
  selectionActive,
  canSelect,
  showStar,
  onToggleStar,
  onSelect,
}: TrackCardProps) {
  const cardContent = (
    <>
      <SvgIcon
        svg={cardCornerSvg}
        className="tarot-corner -top-0.5 -left-0.5"
      />
      <SvgIcon
        svg={cardCornerSvg}
        className="tarot-corner -top-0.5 -right-0.5 -scale-x-100"
      />
      <SvgIcon
        svg={cardCornerSvg}
        className="tarot-corner -bottom-0.5 -left-0.5 -scale-y-100"
      />
      <SvgIcon
        svg={cardCornerSvg}
        className="tarot-corner -bottom-0.5 -right-0.5 -scale-x-100 -scale-y-100"
      />

      <div className="tarot-card-inner">
        {canSelect && (
          <button
            onClick={(e) => {
              e.preventDefault()
              e.stopPropagation()
              onSelect(e, track.uuid, index)
            }}
            className={`absolute bottom-2 right-2 z-10 flex h-5 w-5 cursor-pointer items-center justify-center rounded border transition-colors ${
              isSelected
                ? "border-primary bg-primary text-primary-text"
                : "border-border bg-panel/80 text-transparent hover:border-border-hover hover:text-text-muted"
            }`}
            aria-label={isSelected ? "Deselect track" : "Select track"}
            aria-pressed={isSelected}
          >
            <svg
              viewBox="0 0 16 16"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              className="h-3 w-3"
            >
              <path d="M3 8l3.5 3.5L13 4" />
            </svg>
          </button>
        )}
        {showStar && (
          <button
            onClick={(e) => onToggleStar(e, track.uuid, track.starred ?? false)}
            className="absolute right-3 top-3 z-10 cursor-pointer rounded bg-panel/80 p-1 hover:bg-panel transition-colors"
            aria-label={track.starred ? "Unstar track" : "Star track"}
          >
            <StarIcon
              className={`h-4 w-4 ${track.starred ? "text-star" : "text-text-muted"}`}
            />
          </button>
        )}
        <div className="aspect-square overflow-hidden bg-surface text-track">
          <SvgPreview
            src={`/api/tracks/${track.uuid}/preview.svg?size=256`}
            alt="Track preview"
            className="h-full w-full object-contain"
          />
        </div>
        <div className="px-2.5 pb-2.5">
          <div className="flex items-center gap-1.5">
            <img
              src={`/api/users/${track.user.uuid}/avatar`}
              alt=""
              className="h-4 w-4 shrink-0 rounded-full"
            />
            <p className="truncate font-[Fondamento] text-sm font-medium text-text">
              {track.name}
            </p>
          </div>
          <div className="mt-1 flex items-center gap-2 text-xs text-text-muted">
            <span>{track.user.name}</span>
            <span className="flex items-center gap-0.5">
              <SvgIcon svg={distanceSvg} className="inline h-3 w-3" />
              {formatDistance(track.totalDistanceM)}
            </span>
            <span className="flex items-center gap-0.5">
              <SvgIcon svg={elevationSvg} className="inline h-3 w-3" />
              {formatAscent(track.totalAscentM)}
            </span>
          </div>
          <div className="mt-1.5 overflow-hidden rounded bg-surface text-track">
            <SvgPreview
              src={`/api/tracks/${track.uuid}/profile.svg?size=256`}
              alt="Elevation profile"
              className="w-full"
            />
          </div>
          {track.forecast && (
            <div
              className="mt-1.5 flex items-center gap-x-2 text-xs"
              title={`Forecast: ${new Date(track.forecast.forecastReferenceTime).toLocaleString()}\nStart: ${new Date(track.forecast.startTime).toLocaleString()}`}
            >
              {track.forecast.avgTemperatureC != null && (
                <span className="flex items-center gap-0.5 text-error">
                  <SvgIcon svg={temperatureSvg} className="inline h-3 w-3" />
                  {track.forecast.avgTemperatureC.toFixed(0)}
                  &deg;C
                </span>
              )}
              {track.forecast.totalPrecipitationMm != null && (
                <span className="flex items-center gap-0.5 text-info">
                  <SvgIcon svg={rainSvg} className="inline h-3 w-3" />
                  {track.forecast.totalPrecipitationMm < 0.1
                    ? "dry"
                    : `${track.forecast.totalPrecipitationMm.toFixed(1)} mm`}
                </span>
              )}
              <MiniWindRose
                head={track.forecast.windHeadMs}
                right={track.forecast.windRightMs}
                tail={track.forecast.windTailMs}
                left={track.forecast.windLeftMs}
              />
            </div>
          )}
        </div>
      </div>
    </>
  )

  if (selectionActive && canSelect) {
    return (
      <div
        data-track-card
        onClick={(e) => onSelect(e, track.uuid, index)}
        className={`tarot-card group relative block cursor-pointer ${
          isSelected
            ? "ring-2 ring-primary ring-offset-2 ring-offset-surface"
            : ""
        }`}
        role="checkbox"
        aria-checked={isSelected}
        aria-label={`Select ${track.name}`}
      >
        {cardContent}
      </div>
    )
  }

  return (
    <Link
      data-track-card
      to={`/tracks/${track.uuid}`}
      className={`tarot-card group relative block ${
        isSelected
          ? "ring-2 ring-primary ring-offset-2 ring-offset-surface"
          : ""
      }`}
    >
      {cardContent}
    </Link>
  )
}
