import { useState, useCallback } from "react"
import { useNavigate } from "react-router-dom"
import { useQueryClient } from "@tanstack/react-query"
import { useParams } from "react-router-dom"
import { ArrowsPointingOutIcon } from "@heroicons/react/24/outline"
import { $api } from "../api/client"
import { getTrackColor } from "../lib/trackColor"
import SvgPreview from "../components/SvgPreview"
import { useSession } from "../context/SessionContext"
import StarIcon from "../assets/StarIcon"
import ElevationProfile from "../components/ElevationProfile"
import ForecastChart from "../components/ForecastChart"
import Toast from "../components/Toast"
import useToast from "../hooks/useToast"
import TrackMap from "../components/TrackMap"
import type { RoadClosure } from "../components/TrackMap"
import MapHoverOverlay from "../components/MapHoverOverlay"
import FullscreenMapDialog from "../components/FullscreenMapDialog"
import ForecastControls from "../components/ForecastControls"
import TrackDetails from "../components/TrackDetails"
import TrackEditForm from "../components/TrackEditForm"
import { useHoverStore } from "../hooks/useHoverSync"
import { useForecast } from "../hooks/useForecast"
import PageContainer from "../components/ui/PageContainer"
import Alert from "../components/ui/Alert"

export default function Track() {
  const { uuid } = useParams<{ uuid: string }>()
  const navigate = useNavigate()
  const { user } = useSession()
  const queryClient = useQueryClient()

  const { toast, showToast, dismissToast } = useToast()
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

  const onForecastError = useCallback(
    (msg: string) => showToast(msg),
    [showToast]
  )

  const forecast = useForecast(
    uuid,
    data?.totalDistanceM,
    trackPoints?.length,
    onForecastError
  )

  const starMutation = $api.useMutation("post", "/tracks/{uuid}/star")
  const unstarMutation = $api.useMutation("delete", "/tracks/{uuid}/star")

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
      showToast((err as Error).message)
    }
  }

  if (isLoading) {
    return (
      <PageContainer size="lg">
        <p className="text-text-muted">Loading...</p>
      </PageContainer>
    )
  }

  if (error || !data) {
    return (
      <PageContainer size="lg">
        <p role="alert" className="text-error">
          {(error as Error | null)?.message ?? "Track not found."}
        </p>
      </PageContainer>
    )
  }

  return (
    <PageContainer size="lg">
      {toast && (
        <Toast
          key={toast.key}
          message={toast.message}
          variant={toast.variant}
          onDismiss={dismissToast}
        />
      )}

      <button
        type="button"
        onClick={() => navigate(-1)}
        className="text-sm text-text-muted hover:text-text-secondary transition-colors cursor-pointer"
      >
        &larr; Back
      </button>

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
              forecastTimes={forecast.forecastTimes}
            />
            <button
              type="button"
              onClick={() => setMapFullscreen(true)}
              className="absolute top-2 right-2 z-10 cursor-pointer rounded bg-panel/90 p-1.5 text-text-secondary shadow-sm hover:bg-panel hover:text-text transition-colors"
              aria-label="Fullscreen map"
            >
              <ArrowsPointingOutIcon className="h-5 w-5" />
            </button>

            <FullscreenMapDialog
              open={mapFullscreen}
              onClose={() => setMapFullscreen(false)}
              trackPoints={trackPoints}
              hoverStore={hoverStore}
              color={trackColor}
              closures={closures}
              forecastTimes={forecast.forecastTimes}
            />
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
            Road closures or diversions on this track - see map.
          </p>
        </Alert>
      )}

      {forecast.sunEvents.length > 0 && (
        <Alert variant="warning" className="mt-3">
          <p className="font-medium">
            Don't forget to bring lights when riding in the dark.
          </p>
        </Alert>
      )}

      <ForecastControls
        startHoursOffset={forecast.startHoursOffset}
        speedKmh={forecast.speedKmh}
        estDurationH={forecast.estDurationH}
        forecastLoading={forecast.forecastLoading}
        forecastStatus={forecast.forecastStatus}
        getStartTime={forecast.getStartTime}
        onChangeStart={(h) => {
          forecast.setStartHoursOffset(h)
          forecast.fetchForecast(h, forecast.speedKmh)
        }}
        onChangeSpeed={(s) => {
          forecast.setSpeedKmh(s)
          forecast.fetchForecast(forecast.startHoursOffset, s)
        }}
      />

      {trackPoints && trackPoints.length > 0 && (
        <ElevationProfile
          points={trackPoints}
          hoverStore={hoverStore}
          color={trackColor}
          forecastTimes={forecast.forecastTimes}
        />
      )}

      {forecast.forecastPoints && (
        <ForecastChart
          points={forecast.forecastPoints}
          units={forecast.forecastUnits}
          hoverStore={hoverStore}
          attribution={forecast.forecastAttribution}
          sunEvents={forecast.sunEvents}
        />
      )}

      <TrackDetails track={data} />

      {data.isOwner && (
        <TrackEditForm
          track={data}
          onError={(msg) => showToast(msg)}
          onSuccess={(msg) => showToast(msg, "success")}
        />
      )}
    </PageContainer>
  )
}
