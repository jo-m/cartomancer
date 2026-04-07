import { useState, useMemo, useEffect, useCallback, Fragment } from "react"
import { Link, useNavigate, useParams } from "react-router-dom"
import { useQueryClient } from "@tanstack/react-query"
import { useForm, Controller } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import {
  Dialog,
  DialogPanel,
  Transition,
  TransitionChild,
} from "@headlessui/react"
import { ArrowsPointingOutIcon, XMarkIcon } from "@heroicons/react/24/outline"
import { $api, fetchClient } from "../api/client"
import { getTrackColor } from "../lib/trackColor"
import SvgPreview from "../components/SvgPreview"
import { useSession } from "../context/SessionContext"
import { externalUrl } from "../lib/externalUrl"
import StarIcon from "../assets/StarIcon"
import SvgIcon from "../assets/SvgIcon"
import distanceSvg from "../assets/distance.svg?raw"
import elevationSvg from "../assets/elevation.svg?raw"
import ElevationProfile from "../components/ElevationProfile"
import ForecastChart from "../components/ForecastChart"
import type {
  ForecastPoint,
  ForecastUnits,
  SunEvent,
} from "../components/ForecastChart"
import TagsInput from "../components/TagsInput"
import Toast from "../components/Toast"
import TrackMap from "../components/TrackMap"
import type { RoadClosure } from "../components/TrackMap"
import { useHoverStore, useHoverValue } from "../hooks/useHoverSync"
import { fmtElapsed, fmtClock, buildForecastTimes } from "../lib/time"
import {
  SPORT_LABELS,
  SUB_SPORT_LABELS,
  SUB_SPORTS_BY_SPORT,
} from "../lib/sports"
import PageContainer from "../components/ui/PageContainer"
import Button from "../components/ui/Button"
import Input from "../components/ui/Input"
import Select from "../components/ui/Select"
import Badge from "../components/ui/Badge"
import SectionHeading from "../components/ui/SectionHeading"
import Alert from "../components/ui/Alert"

const TRACK_TYPE_LABELS: Record<number, string> = {
  0: "Unknown",
  1: "Planned",
  2: "Recorded",
}

const FILE_FORMAT_LABELS: Record<number, string> = {
  0: "GPX",
  1: "FIT",
}

const START_HOUR_OPTIONS = [1, 2, 5, 10, 20]
const SPEED_OPTIONS = [20, 25, 28, 30]

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

function getStartTime(hoursOffset: number): Date {
  const startDate = new Date()
  startDate.setHours(startDate.getHours() + hoursOffset)
  return startDate
}

const editSchema = z.object({
  name: z.string().min(1, "Name is required"),
  public: z.boolean(),
  trackType: z.number().int(),
  sport: z.number().int(),
  subSport: z.number().int(),
  tags: z.array(z.string()),
})

type EditFormValues = z.infer<typeof editSchema>

export default function Track() {
  const { uuid } = useParams<{ uuid: string }>()
  const { user } = useSession()
  const queryClient = useQueryClient()
  const navigate = useNavigate()

  const [toastMessage, setToastMessage] = useState<string | null>(null)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [mapFullscreen, setMapFullscreen] = useState(false)
  const hoverStore = useHoverStore()

  const trackColor = getTrackColor()

  const { data, isLoading, error } = $api.useQuery("get", "/tracks/{uuid}", {
    params: { path: { uuid: uuid! } },
  })

  const { data: pointsData } = $api.useQuery("get", "/tracks/{uuid}/points", {
    params: { path: { uuid: uuid! } },
  })

  const trackPoints = pointsData?.points as
    | { lat: number; lon: number; ele: number; d: number }[]
    | undefined

  const { data: closuresData } = $api.useQuery(
    "get",
    "/tracks/{uuid}/road-closures",
    { params: { path: { uuid: uuid! } } }
  )

  const closures = closuresData?.closures as RoadClosure[] | undefined

  const [forecastPoints, setForecastPoints] = useState<ForecastPoint[] | null>(
    null
  )
  const [sunEvents, setSunEvents] = useState<SunEvent[]>([])
  const [forecastLoading, setForecastLoading] = useState(false)
  const [forecastStatus, setForecastStatus] = useState<string | null>(null)
  const [forecastAttribution, setForecastAttribution] = useState<
    { text: string; href: string } | undefined
  >(undefined)
  const [forecastUnits, setForecastUnits] = useState<ForecastUnits>({
    temperatureC: "C",
    precipitationRate: "mm/h",
    windSpeedMs: "m/s",
    windDirectionDeg: "deg",
    relativeWindDirectionDeg: "deg",
  })
  const [startHoursOffset, setStartHoursOffset] = useState(2)
  const [speedKmh, setSpeedKmh] = useState(28)

  const forecastTimes = useMemo(() => {
    if (!forecastPoints || !trackPoints) return undefined
    return buildForecastTimes(forecastPoints, trackPoints.length)
  }, [forecastPoints, trackPoints])

  const estDurationH =
    data && speedKmh > 0 ? data.totalDistanceM / 1000 / speedKmh : 0

  /** Fetches forecast data from the API with the given parameters. */
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
        setToastMessage((err as Error).message)
      } finally {
        setForecastLoading(false)
      }
    },
    [uuid]
  )

  useEffect(() => {
    if (uuid) {
      fetchForecast(startHoursOffset, speedKmh)
    }
  }, [uuid]) // eslint-disable-line react-hooks/exhaustive-deps

  const starMutation = $api.useMutation("post", "/tracks/{uuid}/star")
  const unstarMutation = $api.useMutation("delete", "/tracks/{uuid}/star")
  const editMutation = $api.useMutation("patch", "/tracks/{uuid}")
  const deleteMutation = $api.useMutation("delete", "/tracks/{uuid}")

  const {
    register,
    handleSubmit,
    control,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<EditFormValues>({
    resolver: zodResolver(editSchema),
    values: data?.isOwner
      ? {
          name: data.name,
          public: data.public ?? false,
          trackType: data.trackType,
          sport: data.sport,
          subSport: data.subSport,
          tags: data.tags,
        }
      : undefined,
  })

  const watchedSport = watch("sport")

  async function toggleStar() {
    if (!data) return
    try {
      if (data.starred) {
        await unstarMutation.mutateAsync({
          params: { path: { uuid: data.uuid } },
        })
      } else {
        await starMutation.mutateAsync({
          params: { path: { uuid: data.uuid } },
        })
      }
      await queryClient.invalidateQueries({
        queryKey: ["get", "/tracks/{uuid}"],
      })
    } catch (err) {
      setToastMessage((err as Error).message)
    }
  }

  async function onSubmit(values: EditFormValues) {
    if (!data) return
    try {
      await editMutation.mutateAsync({
        params: { path: { uuid: data.uuid } },
        body: {
          name: values.name,
          public: values.public,
          trackType: values.trackType,
          sport: values.sport,
          subSport: values.subSport,
          tags: values.tags,
        },
      })
      await queryClient.invalidateQueries({
        queryKey: ["get", "/tracks/{uuid}"],
      })
    } catch (err) {
      setToastMessage((err as Error).message)
    }
  }

  async function handleDelete() {
    if (!data) return
    try {
      await deleteMutation.mutateAsync({
        params: { path: { uuid: data.uuid } },
      })
      navigate("/")
    } catch (err) {
      setToastMessage((err as Error).message)
      setConfirmDelete(false)
    }
  }

  if (isLoading) {
    return (
      <PageContainer size="lg" className="py-10">
        <p className="text-text-muted">Loading...</p>
      </PageContainer>
    )
  }

  if (error || !data) {
    return (
      <PageContainer size="lg" className="py-10">
        <p role="alert" className="text-error">
          {(error as Error | null)?.message ?? "Track not found."}
        </p>
      </PageContainer>
    )
  }

  return (
    <PageContainer size="lg" className="py-10">
      {toastMessage && (
        <Toast message={toastMessage} onDismiss={() => setToastMessage(null)} />
      )}

      <Link
        to="/"
        className="text-sm text-text-muted hover:text-text-secondary transition-colors"
      >
        &larr; Tracks
      </Link>

      <div className="mt-4 flex items-center gap-2">
        <img
          src={`/api/users/${data.user.uuid}/avatar`}
          alt=""
          className="h-6 w-6 shrink-0 rounded-full"
        />
        <span className="text-sm text-text-muted">{data.user.name}</span>
      </div>

      <div className="mt-2 flex items-start justify-between gap-4">
        <h1 className="text-2xl font-bold text-text">{data.name}</h1>
        {user && (
          <button
            onClick={toggleStar}
            disabled={starMutation.isPending || unstarMutation.isPending}
            className="shrink-0 cursor-pointer rounded border border-border p-1.5 hover:bg-surface disabled:cursor-not-allowed disabled:opacity-50 transition-colors"
            aria-label={data.starred ? "Unstar track" : "Star track"}
          >
            <StarIcon
              className={`h-5 w-5 ${data.starred ? "text-star" : "text-text-muted"}`}
            />
          </button>
        )}
      </div>

      {data.geonameLabel && (
        <p className="mt-1 text-sm text-text-muted">
          {data.geonameLabel}
          <span className="ml-1.5 text-xs text-text-muted">
            (
            <a
              href="https://www.geonames.org/"
              className="hover:text-text-secondary transition-colors"
              target="_blank"
              rel="noopener noreferrer"
            >
              GeoNames CC-BY 4.0
            </a>
            )
          </span>
        </p>
      )}

      {data.description && (
        <p className="mt-2 text-sm text-text-secondary">{data.description}</p>
      )}

      <div className="mt-6">
        {trackPoints && trackPoints.length > 0 ? (
          <div className="relative">
            <TrackMap
              points={trackPoints}
              hoverStore={hoverStore}
              color={trackColor}
              closures={closures}
            />
            <MapHoverOverlay
              hoverStore={hoverStore}
              trackPoints={trackPoints}
              forecastTimes={forecastTimes}
            />
            <button
              type="button"
              onClick={() => setMapFullscreen(true)}
              className="absolute top-2 right-2 z-10 cursor-pointer rounded bg-panel/90 p-1.5 text-text-secondary shadow-sm hover:bg-panel hover:text-text transition-colors"
              aria-label="Fullscreen map"
            >
              <ArrowsPointingOutIcon className="h-5 w-5" />
            </button>

            <Transition show={mapFullscreen} as={Fragment}>
              <Dialog
                onClose={() => setMapFullscreen(false)}
                className="relative z-50"
              >
                <TransitionChild
                  as={Fragment}
                  enter="ease-out duration-200"
                  enterFrom="opacity-0"
                  enterTo="opacity-100"
                  leave="ease-in duration-150"
                  leaveFrom="opacity-100"
                  leaveTo="opacity-0"
                >
                  <div className="fixed inset-0 bg-overlay" />
                </TransitionChild>
                <TransitionChild
                  as={Fragment}
                  enter="ease-out duration-200"
                  enterFrom="opacity-0 scale-95"
                  enterTo="opacity-100 scale-100"
                  leave="ease-in duration-150"
                  leaveFrom="opacity-100 scale-100"
                  leaveTo="opacity-0 scale-95"
                >
                  <DialogPanel className="fixed inset-0 flex flex-col bg-panel">
                    <button
                      type="button"
                      onClick={() => setMapFullscreen(false)}
                      className="absolute top-3 right-3 z-10 cursor-pointer rounded bg-panel/90 p-1.5 text-text-secondary shadow-sm hover:bg-panel hover:text-text transition-colors"
                      aria-label="Close fullscreen"
                    >
                      <XMarkIcon className="h-6 w-6" />
                    </button>
                    <div className="relative h-full w-full">
                      <TrackMap
                        points={trackPoints}
                        hoverStore={hoverStore}
                        color={trackColor}
                        className="h-full w-full"
                        closures={closures}
                      />
                      <MapHoverOverlay
                        hoverStore={hoverStore}
                        trackPoints={trackPoints}
                        forecastTimes={forecastTimes}
                      />
                    </div>
                  </DialogPanel>
                </TransitionChild>
              </Dialog>
            </Transition>
          </div>
        ) : (
          <div className="overflow-hidden rounded-lg border border-border bg-surface text-track">
            <SvgPreview
              src={`/api/tracks/${data.uuid}/preview.svg?size=512`}
              alt="Track preview"
              className="w-full object-contain"
            />
          </div>
        )}
      </div>

      {closures && closures.length > 0 && (
        <Alert variant="warning" className="mt-3">
          <p className="font-medium">
            🚧 Road closures or diversions on this track - see map.
          </p>
        </Alert>
      )}

      {sunEvents.length > 0 && (
        <Alert variant="warning" className="mt-3">
          <p className="font-medium">
            🔦 Don't forget to bring lights when riding in the dark.
          </p>
        </Alert>
      )}

      <div className="mt-3">
        <div className="flex flex-wrap items-center gap-3">
          <div className="flex items-center gap-1">
            <span className="text-xs text-text-muted">Start in:</span>
            {START_HOUR_OPTIONS.map((h) => (
              <button
                key={h}
                type="button"
                onClick={() => {
                  setStartHoursOffset(h)
                  fetchForecast(h, speedKmh)
                }}
                className={`cursor-pointer rounded border px-1.5 py-1 text-xs transition-colors ${
                  startHoursOffset === h
                    ? "border-border-hover bg-surface font-medium text-text"
                    : "border-border text-text-secondary hover:bg-surface"
                }`}
              >
                +{h}h
              </button>
            ))}
          </div>
          <div className="flex items-center gap-1">
            <span className="text-xs text-text-muted">Est. speed:</span>
            {SPEED_OPTIONS.map((s) => (
              <button
                key={s}
                type="button"
                onClick={() => {
                  setSpeedKmh(s)
                  fetchForecast(startHoursOffset, s)
                }}
                className={`cursor-pointer rounded border px-1.5 py-1 text-xs transition-colors ${
                  speedKmh === s
                    ? "border-border-hover bg-surface font-medium text-text"
                    : "border-border text-text-secondary hover:bg-surface"
                }`}
              >
                {s}km/h
              </button>
            ))}
          </div>
          {forecastLoading && (
            <span className="text-xs text-text-muted">Loading...</span>
          )}
          {!forecastLoading && estDurationH > 0 && startHoursOffset > 0 && (
            <span className="text-xs text-text-muted">
              Est. {estDurationH.toFixed(1)}h,&ensp;
              {fmtClock(getStartTime(startHoursOffset).getTime())}-
              {fmtClock(
                getStartTime(startHoursOffset + estDurationH).getTime()
              )}
            </span>
          )}
        </div>
        {forecastStatus === "none" && (
          <Alert variant="warning" className="mt-2 text-xs">
            No weather forecast data available. Time estimates are still shown.
          </Alert>
        )}
        {forecastStatus === "partial" && (
          <Alert variant="warning" className="mt-2 text-xs">
            Weather forecast only partially covers the requested time window.
          </Alert>
        )}
      </div>

      {trackPoints && trackPoints.length > 0 && (
        <ElevationProfile
          points={trackPoints}
          hoverStore={hoverStore}
          color={trackColor}
          forecastTimes={forecastTimes}
        />
      )}

      {forecastPoints && (
        <ForecastChart
          points={forecastPoints}
          units={forecastUnits}
          hoverStore={hoverStore}
          attribution={forecastAttribution}
          sunEvents={sunEvents}
        />
      )}

      <dl className="mt-6 grid grid-cols-2 gap-x-8 gap-y-4 sm:grid-cols-3">
        <div>
          <dt className="flex items-center gap-1 text-xs font-medium uppercase tracking-wide text-text-muted">
            <SvgIcon svg={distanceSvg} className="h-3.5 w-3.5" />
            Distance
          </dt>
          <dd className="mt-1 text-sm text-text">
            {formatDistance(data.totalDistanceM)}
          </dd>
        </div>
        <div>
          <dt className="flex items-center gap-1 text-xs font-medium uppercase tracking-wide text-text-muted">
            <SvgIcon svg={elevationSvg} className="h-3.5 w-3.5" />
            Ascent
          </dt>
          <dd className="mt-1 text-sm text-text">
            {formatAscent(data.totalAscentM)}
          </dd>
        </div>
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-text-muted">
            Sport
          </dt>
          <dd className="mt-1 text-sm text-text">
            {SPORT_LABELS[data.sport] ?? data.sport}
            {data.subSport !== 0 && (
              <span className="ml-1 text-text-muted">
                ({SUB_SPORT_LABELS[data.subSport] ?? data.subSport})
              </span>
            )}
          </dd>
        </div>
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-text-muted">
            Type
          </dt>
          <dd className="mt-1 text-sm text-text">
            {TRACK_TYPE_LABELS[data.trackType] ?? data.trackType}
          </dd>
        </div>
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-text-muted">
            Format
          </dt>
          <dd className="mt-1 text-sm text-text">
            {FILE_FORMAT_LABELS[data.fileFormat] ?? data.fileFormat}
          </dd>
        </div>
        {data.originalCreatedAt && (
          <div>
            <dt className="text-xs font-medium uppercase tracking-wide text-text-muted">
              Activity date
            </dt>
            <dd className="mt-1 text-sm text-text">
              {formatDate(data.originalCreatedAt)}
            </dd>
          </div>
        )}
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-text-muted">
            Uploaded
          </dt>
          <dd className="mt-1 text-sm text-text">
            {formatDate(data.createdAt)}
          </dd>
        </div>
        {data.author && (
          <div className="col-span-2 sm:col-span-3">
            <dt className="text-xs font-medium uppercase tracking-wide text-text-muted">
              Author
            </dt>
            <dd className="mt-1 text-sm text-text">
              {data.authorLinkUrl ? (
                <a
                  href={externalUrl(data.authorLinkUrl)}
                  className="text-text-secondary hover:text-text underline transition-colors"
                >
                  {data.author}
                </a>
              ) : (
                data.author
              )}
            </dd>
          </div>
        )}
        {data.linkUrl && (
          <div className="col-span-2 sm:col-span-3">
            <dt className="text-xs font-medium uppercase tracking-wide text-text-muted">
              Link
            </dt>
            <dd className="mt-1 text-sm truncate">
              <a
                href={externalUrl(data.linkUrl)}
                className="text-text-secondary hover:text-text underline text-sm transition-colors"
              >
                {data.linkUrl}
              </a>
            </dd>
          </div>
        )}
      </dl>

      {data.tags.length > 0 && (
        <div className="mt-6">
          <SectionHeading>Tags</SectionHeading>
          <div className="mt-2 flex flex-wrap gap-1.5">
            {data.tags.map((tag) => (
              <Badge key={tag}>{tag}</Badge>
            ))}
          </div>
        </div>
      )}

      {data.similarTracks.length > 0 && (
        <div className="mt-6">
          <SectionHeading>Similar tracks</SectionHeading>
          <ul className="mt-2 space-y-1">
            {data.similarTracks.map((st) => (
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
          href={`/api/tracks/${data.uuid}/download`}
          className="text-sm text-text-muted hover:text-text-secondary transition-colors"
        >
          Download original file
        </a>
      </div>

      {data.isOwner && (
        <div className="mt-8 border-t border-border pt-6">
          <h2 className="text-sm font-medium uppercase tracking-wide text-text-muted">
            Edit
          </h2>
          <form onSubmit={handleSubmit(onSubmit)} className="mt-4 space-y-4">
            <Input
              label="Name"
              error={errors.name?.message}
              {...register("name")}
            />

            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  id="track-public"
                  {...register("public")}
                  className="rounded border-border accent-primary"
                />
                <label
                  htmlFor="track-public"
                  className="text-sm text-text-secondary"
                >
                  Public
                </label>
              </div>
              <div className="flex items-center gap-2">
                <label className="text-sm text-text-secondary">Type</label>
                <Select
                  {...register("trackType", { valueAsNumber: true })}
                  className="px-2 py-1 text-sm"
                >
                  <option value={2}>Recorded</option>
                  <option value={1}>Planned</option>
                </Select>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <Select
                label="Sport"
                {...register("sport", { valueAsNumber: true })}
              >
                {Object.entries(SPORT_LABELS).map(([k, v]) => (
                  <option key={k} value={k}>
                    {v}
                  </option>
                ))}
              </Select>
              <Select
                label="Sub-sport"
                {...register("subSport", { valueAsNumber: true })}
              >
                {(SUB_SPORTS_BY_SPORT[watchedSport] ?? [0]).map((id) => (
                  <option key={id} value={id}>
                    {SUB_SPORT_LABELS[id]}
                  </option>
                ))}
              </Select>
            </div>

            <div>
              <label className="block text-xs font-medium text-text-secondary">
                Tags
              </label>
              <div className="mt-1">
                <Controller
                  name="tags"
                  control={control}
                  render={({ field }) => (
                    <TagsInput
                      value={field.value ?? []}
                      onChange={field.onChange}
                    />
                  )}
                />
              </div>
            </div>

            <div className="flex items-center gap-3 pt-1">
              <Button
                type="submit"
                variant="secondary"
                disabled={isSubmitting || editMutation.isPending}
              >
                {editMutation.isPending ? "Saving..." : "Save"}
              </Button>

              {editMutation.isSuccess && (
                <span className="text-xs text-text-muted">Saved.</span>
              )}

              <div className="ml-auto flex items-center gap-2">
                {confirmDelete ? (
                  <>
                    <span className="text-sm text-text-secondary">
                      Delete this track?
                    </span>
                    <Button
                      variant="danger"
                      onClick={handleDelete}
                      disabled={deleteMutation.isPending}
                    >
                      {deleteMutation.isPending ? "Deleting..." : "Confirm"}
                    </Button>
                    <Button
                      variant="secondary"
                      onClick={() => setConfirmDelete(false)}
                    >
                      Cancel
                    </Button>
                  </>
                ) : (
                  <Button
                    variant="secondary"
                    onClick={() => setConfirmDelete(true)}
                  >
                    Delete
                  </Button>
                )}
              </div>
            </div>
          </form>
        </div>
      )}
    </PageContainer>
  )
}

/** Overlay on the map showing hover info with forecast timing. */
function MapHoverOverlay({
  hoverStore,
  trackPoints,
  forecastTimes,
}: {
  hoverStore: ReturnType<typeof useHoverStore>
  trackPoints: { lat: number; lon: number; ele: number; d: number }[]
  forecastTimes?: number[]
}) {
  const hoverIndex = useHoverValue(hoverStore)

  if (hoverIndex == null || hoverIndex < 0 || hoverIndex >= trackPoints.length)
    return null

  const p = trackPoints[hoverIndex]
  const dKm = (p.d / 1000).toFixed(1)

  let timeInfo = ""
  if (forecastTimes && forecastTimes.length > hoverIndex) {
    const ts = forecastTimes[hoverIndex]
    const startTs = forecastTimes[0]
    timeInfo = ` · +${fmtElapsed(ts - startTs)} · ${fmtClock(ts)}`
  }

  return (
    <div className="pointer-events-none absolute bottom-2 left-2 rounded bg-panel/90 px-2 py-1 text-xs text-text-secondary shadow-sm">
      {dKm} km &middot; {Math.round(p.ele)} m{timeInfo}
    </div>
  )
}
