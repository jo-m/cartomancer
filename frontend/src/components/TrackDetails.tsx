import { Link } from "react-router-dom"
import { externalUrl } from "../lib/externalUrl"
import { SPORT_LABELS, SUB_SPORT_LABELS } from "../lib/sports"
import SvgIcon from "../assets/SvgIcon"
import distanceSvg from "../assets/distance.svg?raw"
import elevationSvg from "../assets/elevation.svg?raw"
import SectionHeading from "./ui/SectionHeading"
import Badge from "./ui/Badge"

const TRACK_TYPE_LABELS: Record<number, string> = {
  0: "Unknown",
  1: "Planned",
  2: "Recorded",
}

const FILE_FORMAT_LABELS: Record<number, string> = {
  0: "GPX",
  1: "FIT",
}

function formatDistance(m: number): string {
  return `${(m / 1000).toFixed(1)} km`
}

function formatAscent(m: number): string {
  return `${Math.round(m)} m`
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  })
}

interface SimilarTrack {
  uuid: string
  name: string
  totalDistanceM: number
}

interface TrackDetailData {
  uuid: string
  totalDistanceM: number
  totalAscentM: number
  sport: number
  subSport: number
  trackType: number
  fileFormat: number
  originalCreatedAt?: string
  createdAt: string
  author?: string
  authorLinkUrl?: string
  linkUrl?: string
  tags: string[]
  similarTracks: SimilarTrack[]
}

export interface TrackDetailsProps {
  track: TrackDetailData
}

/** Displays track metadata (stats, dates, tags, similar tracks, download link). */
export default function TrackDetails({ track }: TrackDetailsProps) {
  return (
    <>
      <dl className="mt-6 grid grid-cols-2 gap-x-8 gap-y-4 sm:grid-cols-3">
        <div>
          <dt className="flex items-center gap-1 text-xs font-medium uppercase tracking-wide text-text-muted">
            <SvgIcon svg={distanceSvg} className="h-3.5 w-3.5" />
            Distance
          </dt>
          <dd className="mt-1 text-sm text-text">
            {formatDistance(track.totalDistanceM)}
          </dd>
        </div>
        <div>
          <dt className="flex items-center gap-1 text-xs font-medium uppercase tracking-wide text-text-muted">
            <SvgIcon svg={elevationSvg} className="h-3.5 w-3.5" />
            Ascent
          </dt>
          <dd className="mt-1 text-sm text-text">
            {formatAscent(track.totalAscentM)}
          </dd>
        </div>
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-text-muted">
            Sport
          </dt>
          <dd className="mt-1 text-sm text-text">
            {SPORT_LABELS[track.sport] ?? track.sport}
            {track.subSport !== 0 && (
              <span className="ml-1 text-text-muted">
                ({SUB_SPORT_LABELS[track.subSport] ?? track.subSport})
              </span>
            )}
          </dd>
        </div>
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-text-muted">
            Type
          </dt>
          <dd className="mt-1 text-sm text-text">
            {TRACK_TYPE_LABELS[track.trackType] ?? track.trackType}
          </dd>
        </div>
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-text-muted">
            Format
          </dt>
          <dd className="mt-1 text-sm text-text">
            {FILE_FORMAT_LABELS[track.fileFormat] ?? track.fileFormat}
          </dd>
        </div>
        {track.originalCreatedAt && (
          <div>
            <dt className="text-xs font-medium uppercase tracking-wide text-text-muted">
              Activity date
            </dt>
            <dd className="mt-1 text-sm text-text">
              {formatDate(track.originalCreatedAt)}
            </dd>
          </div>
        )}
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-text-muted">
            Uploaded
          </dt>
          <dd className="mt-1 text-sm text-text">
            {formatDate(track.createdAt)}
          </dd>
        </div>
        {track.author && (
          <div className="col-span-2 sm:col-span-3">
            <dt className="text-xs font-medium uppercase tracking-wide text-text-muted">
              Author
            </dt>
            <dd className="mt-1 text-sm text-text">
              {track.authorLinkUrl ? (
                <a
                  href={externalUrl(track.authorLinkUrl)}
                  className="text-text-secondary hover:text-text underline transition-colors"
                >
                  {track.author}
                </a>
              ) : (
                track.author
              )}
            </dd>
          </div>
        )}
        {track.linkUrl && (
          <div className="col-span-2 sm:col-span-3">
            <dt className="text-xs font-medium uppercase tracking-wide text-text-muted">
              Link
            </dt>
            <dd className="mt-1 text-sm truncate">
              <a
                href={externalUrl(track.linkUrl)}
                className="text-text-secondary hover:text-text underline text-sm transition-colors"
              >
                {track.linkUrl}
              </a>
            </dd>
          </div>
        )}
      </dl>

      {track.tags.length > 0 && (
        <div className="mt-6">
          <SectionHeading>Tags</SectionHeading>
          <div className="mt-2 flex flex-wrap gap-1.5">
            {track.tags.map((tag) => (
              <Badge key={tag}>{tag}</Badge>
            ))}
          </div>
        </div>
      )}

      {track.similarTracks.length > 0 && (
        <div className="mt-6">
          <SectionHeading>Similar tracks</SectionHeading>
          <ul className="mt-2 space-y-1">
            {track.similarTracks.map((st) => (
              <li key={st.uuid}>
                <Link
                  to={`/tracks/${st.uuid}`}
                  className="text-sm text-text-secondary hover:text-text transition-colors"
                >
                  {st.name}
                  <span className="ml-1.5 text-text-muted">
                    {formatDistance(st.totalDistanceM)}
                  </span>
                </Link>
              </li>
            ))}
          </ul>
        </div>
      )}

      <div className="mt-6">
        <a
          href={`/api/tracks/${track.uuid}/download`}
          className="text-sm text-text-muted hover:text-text-secondary transition-colors"
        >
          Download original file
        </a>
      </div>
    </>
  )
}
