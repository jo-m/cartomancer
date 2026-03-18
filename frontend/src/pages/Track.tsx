import {
  useState,
  useMemo,
  useEffect,
  useCallback,
  useRef,
  Fragment,
} from "react"
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
import { useSession } from "../context/SessionContext"
import StarIcon from "../assets/StarIcon"
import ElevationProfile from "../components/ElevationProfile"
import ForecastChart from "../components/ForecastChart"
import type { ForecastPoint, ForecastUnits } from "../components/ForecastChart"
import TagsInput from "../components/TagsInput"
import Toast from "../components/Toast"
import TrackMap from "../components/TrackMap"
import { useHoverStore, useHoverValue } from "../hooks/useHoverSync"
import { fmtElapsed, fmtClock, buildForecastTimes } from "../lib/time"
import {
  SPORT_LABELS,
  SUB_SPORT_LABELS,
  SUB_SPORTS_BY_SPORT,
} from "../lib/sports"

const TRACK_TYPE_LABELS: Record<number, string> = {
  0: "Unknown",
  1: "Planned",
  2: "Recorded",
}

const FILE_FORMAT_LABELS: Record<number, string> = {
  0: "GPX",
  1: "FIT",
}

const HOVER_DEBOUNCE_MS = 100
const START_HOUR_OPTIONS = [1, 2, 5, 12]
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

const editSchema = z.object({
  name: z.string().min(1, "Name is required"),
  public: z.boolean(),
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

  const { data: appConfig } = $api.useQuery("get", "/app_config")
  const trackColor =
    appConfig?.trackColor && appConfig.trackColor !== "currentColor"
      ? appConfig.trackColor
      : "#111827"

  const { data, isLoading, error } = $api.useQuery("get", "/tracks/{uuid}", {
    params: { path: { uuid: uuid! } },
  })

  const { data: pointsData } = $api.useQuery("get", "/tracks/{uuid}/points", {
    params: { path: { uuid: uuid! } },
  })

  const trackPoints = pointsData?.points as
    | { lat: number; lon: number; ele: number; d: number }[]
    | undefined

  // Forecast state.
  const [forecastPoints, setForecastPoints] = useState<ForecastPoint[] | null>(
    null
  )
  const [forecastLoading, setForecastLoading] = useState(false)
  const [forecastStatus, setForecastStatus] = useState<string | null>(null)
  const [forecastAttribution, setForecastAttribution] = useState("")
  const [forecastAttributionHref, setForecastAttributionHref] = useState("")
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
      const startDate = new Date()
      startDate.setMinutes(0, 0, 0)
      startDate.setHours(startDate.getHours() + hoursOffset)

      setForecastLoading(true)
      setForecastStatus(null)
      try {
        const { data: result, error: apiError } = await fetchClient.POST(
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
        setForecastAttribution(result.attribution ?? "")
        setForecastAttributionHref(result.attributionHref ?? "")
        if (result.units) {
          setForecastUnits(result.units as ForecastUnits)
        }
        setForecastPoints(result.points as ForecastPoint[])
      } catch (err) {
        setToastMessage((err as Error).message)
      } finally {
        setForecastLoading(false)
      }
    },
    [uuid]
  )

  // Auto-load forecast on first render.
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
          sport: values.sport,
          subSport: values.subSport,
          tags: values.tags,
          trackType: data.trackType,
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
      <div className="mx-auto max-w-3xl px-4 py-10">
        <p className="text-gray-500">Loading...</p>
      </div>
    )
  }

  if (error || !data) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-10">
        <p className="text-red-600">
          {(error as Error | null)?.message ?? "Track not found."}
        </p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-3xl px-4 py-10">
      {toastMessage && (
        <Toast message={toastMessage} onDismiss={() => setToastMessage(null)} />
      )}

      <Link to="/" className="text-sm text-gray-500 hover:text-gray-700">
        ← Tracks
      </Link>

      <div className="mt-4 flex items-center gap-2">
        <img
          src={`/api/users/${data.userUuid}/avatar`}
          alt=""
          className="h-6 w-6 shrink-0 rounded-full"
        />
        <span className="text-sm text-gray-500">{data.userName}</span>
      </div>

      <div className="mt-2 flex items-start justify-between gap-4">
        <h1 className="text-2xl font-bold text-gray-900">{data.name}</h1>
        {user && (
          <button
            onClick={toggleStar}
            disabled={starMutation.isPending || unstarMutation.isPending}
            className="shrink-0 cursor-pointer rounded border border-gray-200 p-1.5 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <StarIcon
              className={`h-5 w-5 ${data.starred ? "text-yellow-400" : "text-gray-300"}`}
            />
          </button>
        )}
      </div>

      {data.geonameLabel && (
        <p className="mt-1 text-sm text-gray-500">
          {data.geonameLabel}
          <span className="ml-1.5 text-xs text-gray-400">
            (
            <a
              href="https://www.geonames.org/"
              className="hover:text-gray-600"
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
        <p className="mt-2 text-sm text-gray-600">{data.description}</p>
      )}

      <div className="mt-6">
        {trackPoints && trackPoints.length > 0 ? (
          <div className="relative">
            <TrackMap
              points={trackPoints}
              hoverStore={hoverStore}
              color={trackColor}
            />
            <MapHoverOverlay
              hoverStore={hoverStore}
              trackPoints={trackPoints}
              forecastTimes={forecastTimes}
            />
            <button
              type="button"
              onClick={() => setMapFullscreen(true)}
              className="absolute top-2 right-2 z-10 rounded bg-white/90 p-1.5 text-gray-600 shadow-sm hover:bg-white hover:text-gray-900"
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
                  <div className="fixed inset-0 bg-black/40" />
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
                  <DialogPanel className="fixed inset-0 flex flex-col bg-white">
                    <button
                      type="button"
                      onClick={() => setMapFullscreen(false)}
                      className="absolute top-3 right-3 z-10 rounded bg-white/90 p-1.5 text-gray-600 shadow-sm hover:bg-white hover:text-gray-900"
                    >
                      <XMarkIcon className="h-6 w-6" />
                    </button>
                    <div className="relative h-full w-full">
                      <TrackMap
                        points={trackPoints}
                        hoverStore={hoverStore}
                        color={trackColor}
                        className="h-full w-full"
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
          <div className="overflow-hidden rounded-lg border border-gray-200 bg-gray-50">
            <img
              src={`/api/tracks/${data.uuid}/preview.svg?size=512`}
              alt="Track preview"
              className="w-full object-contain"
            />
          </div>
        )}
      </div>

      <div className="mt-3">
        <div className="flex flex-wrap items-center gap-3">
          <div className="flex items-center gap-1">
            <span className="text-xs text-gray-500">Start in:</span>
            {START_HOUR_OPTIONS.map((h) => (
              <button
                key={h}
                type="button"
                onClick={() => {
                  setStartHoursOffset(h)
                  fetchForecast(h, speedKmh)
                }}
                className={`rounded border px-1.5 py-1 text-xs ${
                  startHoursOffset === h
                    ? "border-gray-400 bg-gray-200 font-medium text-gray-800"
                    : "border-gray-200 text-gray-600 hover:bg-gray-100"
                }`}
              >
                +{h}h
              </button>
            ))}
          </div>
          <div className="flex items-center gap-1">
            <span className="text-xs text-gray-500">Est. speed:</span>
            {SPEED_OPTIONS.map((s) => (
              <button
                key={s}
                type="button"
                onClick={() => {
                  setSpeedKmh(s)
                  fetchForecast(startHoursOffset, s)
                }}
                className={`rounded border px-1.5 py-1 text-xs ${
                  speedKmh === s
                    ? "border-gray-400 bg-gray-200 font-medium text-gray-800"
                    : "border-gray-200 text-gray-600 hover:bg-gray-100"
                }`}
              >
                {s}km/h
              </button>
            ))}
          </div>
          {forecastLoading && (
            <span className="text-xs text-gray-400">Loading...</span>
          )}
          {!forecastLoading && estDurationH > 0 && (
            <span className="text-xs text-gray-400">
              Est. {estDurationH.toFixed(1)}h
            </span>
          )}
        </div>
        {forecastStatus === "none" && (
          <p className="mt-2 text-xs text-amber-600">
            No weather forecast data available. Time estimates are still shown.
          </p>
        )}
        {forecastStatus === "partial" && (
          <p className="mt-2 text-xs text-amber-600">
            Weather forecast only partially covers the requested time window.
          </p>
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
          attributionHref={forecastAttributionHref}
        />
      )}

      <dl className="mt-6 grid grid-cols-2 gap-x-8 gap-y-4 sm:grid-cols-3">
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-gray-500">
            Distance
          </dt>
          <dd className="mt-1 text-sm text-gray-900">
            {formatDistance(data.totalDistanceM)}
          </dd>
        </div>
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-gray-500">
            Ascent
          </dt>
          <dd className="mt-1 text-sm text-gray-900">
            {formatAscent(data.totalAscentM)}
          </dd>
        </div>
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-gray-500">
            Sport
          </dt>
          <dd className="mt-1 text-sm text-gray-900">
            {SPORT_LABELS[data.sport] ?? data.sport}
            {data.subSport !== 0 && (
              <span className="ml-1 text-gray-500">
                ({SUB_SPORT_LABELS[data.subSport] ?? data.subSport})
              </span>
            )}
          </dd>
        </div>
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-gray-500">
            Type
          </dt>
          <dd className="mt-1 text-sm text-gray-900">
            {TRACK_TYPE_LABELS[data.trackType] ?? data.trackType}
          </dd>
        </div>
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-gray-500">
            Format
          </dt>
          <dd className="mt-1 text-sm text-gray-900">
            {FILE_FORMAT_LABELS[data.fileFormat] ?? data.fileFormat}
          </dd>
        </div>
        {data.originalCreatedAt && (
          <div>
            <dt className="text-xs font-medium uppercase tracking-wide text-gray-500">
              Activity date
            </dt>
            <dd className="mt-1 text-sm text-gray-900">
              {formatDate(data.originalCreatedAt)}
            </dd>
          </div>
        )}
        <div>
          <dt className="text-xs font-medium uppercase tracking-wide text-gray-500">
            Uploaded
          </dt>
          <dd className="mt-1 text-sm text-gray-900">
            {formatDate(data.createdAt)}
          </dd>
        </div>
        {data.source && (
          <div className="col-span-2 sm:col-span-3">
            <dt className="text-xs font-medium uppercase tracking-wide text-gray-500">
              Source
            </dt>
            <dd className="mt-1 text-sm text-gray-900">{data.source}</dd>
          </div>
        )}
      </dl>

      {data.tags.length > 0 && (
        <div className="mt-6">
          <p className="text-xs font-medium uppercase tracking-wide text-gray-500">
            Tags
          </p>
          <div className="mt-2 flex flex-wrap gap-1.5">
            {data.tags.map((tag) => (
              <span
                key={tag}
                className="rounded-full border border-gray-200 bg-gray-100 px-2.5 py-0.5 text-xs text-gray-700"
              >
                {tag}
              </span>
            ))}
          </div>
        </div>
      )}

      <div className="mt-6">
        <a
          href={`/api/tracks/${data.uuid}/download`}
          className="text-sm text-gray-500 hover:text-gray-700"
        >
          Download original file
        </a>
      </div>

      {data.isOwner && (
        <div className="mt-8 border-t border-gray-200 pt-6">
          <h2 className="text-sm font-medium uppercase tracking-wide text-gray-500">
            Edit
          </h2>
          <form onSubmit={handleSubmit(onSubmit)} className="mt-4 space-y-4">
            <div>
              <label className="block text-xs font-medium text-gray-700">
                Name
              </label>
              <input
                {...register("name")}
                className="mt-1 w-full rounded border border-gray-200 px-3 py-1.5 text-sm text-gray-900 focus:outline-none focus:ring-1 focus:ring-gray-300"
              />
              {errors.name && (
                <p className="mt-1 text-xs text-red-600">
                  {errors.name.message}
                </p>
              )}
            </div>

            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id="track-public"
                {...register("public")}
                className="rounded border-gray-300"
              />
              <label htmlFor="track-public" className="text-sm text-gray-700">
                Public
              </label>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-medium text-gray-700">
                  Sport
                </label>
                <select
                  {...register("sport", { valueAsNumber: true })}
                  className="mt-1 w-full rounded border border-gray-200 px-3 py-1.5 text-sm text-gray-900 focus:outline-none focus:ring-1 focus:ring-gray-300"
                >
                  {Object.entries(SPORT_LABELS).map(([k, v]) => (
                    <option key={k} value={k}>
                      {v}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-700">
                  Sub-sport
                </label>
                <select
                  {...register("subSport", { valueAsNumber: true })}
                  className="mt-1 w-full rounded border border-gray-200 px-3 py-1.5 text-sm text-gray-900 focus:outline-none focus:ring-1 focus:ring-gray-300"
                >
                  {(SUB_SPORTS_BY_SPORT[watchedSport] ?? [0]).map((id) => (
                    <option key={id} value={id}>
                      {SUB_SPORT_LABELS[id]}
                    </option>
                  ))}
                </select>
              </div>
            </div>

            <div>
              <label className="block text-xs font-medium text-gray-700">
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
              <button
                type="submit"
                disabled={isSubmitting || editMutation.isPending}
                className="rounded border border-gray-300 bg-white px-4 py-1.5 text-sm text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {editMutation.isPending ? "Saving..." : "Save"}
              </button>

              {editMutation.isSuccess && (
                <span className="text-xs text-gray-500">Saved.</span>
              )}

              <div className="ml-auto flex items-center gap-2">
                {confirmDelete ? (
                  <>
                    <span className="text-sm text-gray-600">
                      Delete this track?
                    </span>
                    <button
                      type="button"
                      onClick={handleDelete}
                      disabled={deleteMutation.isPending}
                      className="rounded border border-red-300 bg-white px-3 py-1.5 text-sm text-red-600 hover:bg-red-50 disabled:opacity-50"
                    >
                      {deleteMutation.isPending ? "Deleting..." : "Confirm"}
                    </button>
                    <button
                      type="button"
                      onClick={() => setConfirmDelete(false)}
                      className="rounded border border-gray-300 bg-white px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50"
                    >
                      Cancel
                    </button>
                  </>
                ) : (
                  <button
                    type="button"
                    onClick={() => setConfirmDelete(true)}
                    className="rounded border border-gray-300 bg-white px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50"
                  >
                    Delete
                  </button>
                )}
              </div>
            </div>
          </form>
        </div>
      )}
    </div>
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
  const rawIndex = useHoverValue(hoverStore)
  const [hoverIndex, setHoverIndex] = useState<number | null>(null)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    if (timerRef.current) clearTimeout(timerRef.current)
    const delay = rawIndex == null ? 0 : HOVER_DEBOUNCE_MS
    timerRef.current = setTimeout(() => setHoverIndex(rawIndex), delay)
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current)
    }
  }, [rawIndex])

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
    <div className="pointer-events-none absolute bottom-2 left-2 rounded bg-white/90 px-2 py-1 text-xs text-gray-700 shadow-sm">
      {dKm} km &middot; {Math.round(p.ele)} m{timeInfo}
    </div>
  )
}
