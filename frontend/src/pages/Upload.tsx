import { useState, useRef, useEffect, useCallback } from "react"
import { Link } from "react-router-dom"
import { useQueryClient } from "@tanstack/react-query"
import { fetchClient, $api } from "../api/client"
import TagsInput from "../components/TagsInput"
import {
  SPORT_LABELS,
  SUB_SPORT_LABELS,
  SUB_SPORTS_BY_SPORT,
} from "../lib/sports"

type UploadStatus = "pending" | "uploading" | "error"

interface FileUpload {
  id: string
  file: File | null
  filename: string
  status: UploadStatus
  errorMsg?: string
}

const SESSION_KEY = "upload_errors"

function loadErrorsFromSession(): FileUpload[] {
  try {
    const raw = sessionStorage.getItem(SESSION_KEY)
    if (!raw) return []
    return JSON.parse(raw) as FileUpload[]
  } catch {
    return []
  }
}

function persistErrorsToSession(uploads: FileUpload[]) {
  const errors = uploads
    .filter((u) => u.status === "error")
    .map(({ id, filename, status, errorMsg }) => ({
      id,
      filename,
      status,
      errorMsg,
      file: null,
    }))
  sessionStorage.setItem(SESSION_KEY, JSON.stringify(errors))
}

async function uploadFile(
  item: FileUpload,
  update: (id: string, patch: Partial<FileUpload>) => void,
  onSuccess: (id: string) => void
) {
  update(item.id, { status: "uploading" })
  try {
    await fetchClient.POST("/tracks", {
      body: { file: item.file! } as never,
      bodySerializer(body) {
        const fd = new FormData()
        fd.append("file", item.file!)
        void body
        return fd
      },
    })
    onSuccess(item.id)
  } catch (e) {
    update(item.id, { status: "error", errorMsg: (e as Error).message })
  }
}

function formatDistance(m: number): string {
  return `${(m / 1000).toFixed(1)} km`
}

function formatAscent(m: number): string {
  return `${Math.round(m)} m`
}

export default function Upload() {
  const queryClient = useQueryClient()
  const [uploads, setUploads] = useState<FileUpload[]>(() =>
    loadErrorsFromSession()
  )
  const [isDragging, setIsDragging] = useState(false)
  const [bulkTags, setBulkTags] = useState<string[]>([])
  const [bulkSport, setBulkSport] = useState("")
  const [bulkSubSport, setBulkSubSport] = useState("")
  const inputRef = useRef<HTMLInputElement>(null)

  const { data: editingData, isLoading: editingLoading } = $api.useQuery(
    "get",
    "/tracks/editing"
  )

  const dismissMutation = $api.useMutation("post", "/tracks/editing-complete")
  const bulkEditMutation = $api.useMutation("patch", "/tracks")

  useEffect(() => {
    persistErrorsToSession(uploads)
  }, [uploads])

  const updateUpload = useCallback((id: string, patch: Partial<FileUpload>) => {
    setUploads((prev) =>
      prev.map((u) => (u.id === id ? { ...u, ...patch } : u))
    )
  }, [])

  const handleSuccess = useCallback(
    (id: string) => {
      setUploads((prev) => prev.filter((u) => u.id !== id))
      void queryClient.invalidateQueries({
        queryKey: ["get", "/tracks/editing"],
      })
    },
    [queryClient]
  )

  function addFiles(files: FileList | File[]) {
    const valid = Array.from(files).filter(
      (f) => f.name.endsWith(".gpx") || f.name.endsWith(".fit")
    )
    if (valid.length === 0) return
    const newItems: FileUpload[] = valid.map((f) => ({
      id: crypto.randomUUID(),
      file: f,
      filename: f.name,
      status: "pending",
    }))
    setUploads((prev) => [...newItems, ...prev])
    newItems.forEach(
      (item) => void uploadFile(item, updateUpload, handleSuccess)
    )
  }

  function dismiss(id: string) {
    setUploads((prev) => prev.filter((u) => u.id !== id))
  }

  function dismissAllErrors() {
    setUploads((prev) =>
      prev.filter((u) => u.status === "pending" || u.status === "uploading")
    )
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault()
    setIsDragging(false)
    addFiles(e.dataTransfer.files)
  }

  function handleDragOver(e: React.DragEvent) {
    e.preventDefault()
    setIsDragging(true)
  }

  function handleDragLeave(e: React.DragEvent) {
    if (!e.currentTarget.contains(e.relatedTarget as Node)) {
      setIsDragging(false)
    }
  }

  function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    if (e.target.files) {
      addFiles(e.target.files)
      e.target.value = ""
    }
  }

  function dismissTrack(uuid: string) {
    dismissMutation.mutate(
      { body: { uuids: [uuid] } },
      {
        onSuccess: () => {
          void queryClient.invalidateQueries({
            queryKey: ["get", "/tracks/editing"],
          })
        },
      }
    )
  }

  function dismissAllTracks() {
    const uuids = pendingTracks.map((t) => t.uuid)
    if (uuids.length === 0) return
    dismissMutation.mutate(
      { body: { uuids } },
      {
        onSuccess: () => {
          void queryClient.invalidateQueries({
            queryKey: ["get", "/tracks/editing"],
          })
        },
      }
    )
  }

  function bulkSetTags() {
    const uuids = pendingTracks.map((t) => t.uuid)
    if (bulkTags.length === 0 || uuids.length === 0) return
    bulkEditMutation.mutate(
      { body: { uuids, tags: bulkTags } },
      {
        onSuccess: () => {
          void queryClient.invalidateQueries({
            queryKey: ["get", "/tracks/editing"],
          })
        },
      }
    )
  }

  function bulkSetVisibility(isPublic: boolean) {
    const uuids = pendingTracks.map((t) => t.uuid)
    if (uuids.length === 0) return
    bulkEditMutation.mutate(
      { body: { uuids, public: isPublic } },
      {
        onSuccess: () => {
          void queryClient.invalidateQueries({
            queryKey: ["get", "/tracks/editing"],
          })
        },
      }
    )
  }

  function bulkSetSport() {
    const uuids = pendingTracks.map((t) => t.uuid)
    if (!bulkSport || uuids.length === 0) return
    const body: Parameters<typeof bulkEditMutation.mutate>[0]["body"] = {
      uuids,
      sport: parseInt(bulkSport),
    }
    if (bulkSubSport !== "") body.subSport = parseInt(bulkSubSport)
    bulkEditMutation.mutate(
      { body },
      {
        onSuccess: () => {
          void queryClient.invalidateQueries({
            queryKey: ["get", "/tracks/editing"],
          })
        },
      }
    )
  }

  const activeUploads = uploads.filter(
    (u) => u.status === "pending" || u.status === "uploading"
  )
  const failedUploads = uploads.filter((u) => u.status === "error")
  const pendingTracks = editingData?.tracks ?? []

  return (
    <div className="mx-auto max-w-2xl px-4 py-10">
      <h1 className="mb-6 text-xl font-semibold text-gray-900">
        Upload Tracks
      </h1>

      <div
        onClick={() => inputRef.current?.click()}
        onDrop={handleDrop}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        className={`cursor-pointer rounded-lg border-2 border-dashed p-12 text-center transition-colors ${
          isDragging
            ? "border-blue-400 bg-blue-50"
            : "border-gray-300 bg-white hover:border-gray-400"
        }`}
      >
        <input
          ref={inputRef}
          type="file"
          multiple
          accept=".gpx,.fit"
          className="hidden"
          onChange={handleChange}
        />
        <p className="text-sm text-gray-500">
          Drop <span className="font-medium text-gray-700">.gpx</span> or{" "}
          <span className="font-medium text-gray-700">.fit</span> files here, or
          click to select
        </p>
      </div>

      {activeUploads.length > 0 && (
        <ul className="mt-4 divide-y divide-gray-100 rounded-lg border border-gray-200 bg-white">
          {activeUploads.map((u) => (
            <li
              key={u.id}
              className="flex items-center gap-3 px-4 py-2.5 text-sm"
            >
              <span className="min-w-0 flex-1 truncate text-gray-800">
                {u.filename}
              </span>
              {u.status === "pending" && (
                <span className="shrink-0 text-gray-400">Pending</span>
              )}
              {u.status === "uploading" && (
                <span className="shrink-0 text-blue-500">Uploading…</span>
              )}
            </li>
          ))}
        </ul>
      )}

      {failedUploads.length > 0 && (
        <div className="mt-4">
          <div className="mb-2 flex items-center justify-between">
            <p className="text-xs font-medium uppercase tracking-wide text-gray-500">
              Failed uploads
            </p>
            {failedUploads.length > 1 && (
              <button
                onClick={dismissAllErrors}
                className="cursor-pointer text-xs text-gray-400 hover:text-gray-600"
              >
                Dismiss all
              </button>
            )}
          </div>
          <ul className="divide-y divide-gray-100 rounded-lg border border-gray-200 bg-white">
            {failedUploads.map((u) => (
              <li
                key={u.id}
                className="flex items-center gap-3 px-4 py-2.5 text-sm"
              >
                <span className="min-w-0 flex-1 truncate text-gray-800">
                  {u.filename}
                </span>
                <span className="shrink-0 text-red-600">{u.errorMsg}</span>
                <button
                  onClick={() => dismiss(u.id)}
                  className="shrink-0 cursor-pointer text-xs text-gray-300 hover:text-gray-500"
                >
                  Dismiss
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}

      {(editingLoading || pendingTracks.length > 0) && (
        <div className="mt-8">
          <div className="mb-3 flex items-center justify-between">
            <p className="text-xs font-medium uppercase tracking-wide text-gray-500">
              Pending review
            </p>
            {pendingTracks.length > 1 && (
              <button
                onClick={dismissAllTracks}
                disabled={dismissMutation.isPending}
                className="cursor-pointer text-xs text-gray-400 hover:text-gray-600 disabled:opacity-50"
              >
                Dismiss all
              </button>
            )}
          </div>
          {pendingTracks.length > 1 && (
            <div className="mb-3 flex flex-wrap items-center gap-2">
              <button
                onClick={() => bulkSetVisibility(true)}
                disabled={bulkEditMutation.isPending}
                className="cursor-pointer text-xs text-gray-500 hover:text-gray-700 disabled:opacity-50"
              >
                Set all public
              </button>
              <button
                onClick={() => bulkSetVisibility(false)}
                disabled={bulkEditMutation.isPending}
                className="cursor-pointer text-xs text-gray-500 hover:text-gray-700 disabled:opacity-50"
              >
                Set all private
              </button>
              <span className="text-gray-200">|</span>
              <select
                value={bulkSport}
                onChange={(e) => {
                  setBulkSport(e.target.value)
                  setBulkSubSport("")
                }}
                className="cursor-pointer rounded border border-gray-200 px-1.5 py-0.5 text-xs text-gray-700 focus:outline-none focus:ring-1 focus:ring-gray-300"
              >
                <option value="">Sport…</option>
                {Object.entries(SPORT_LABELS).map(([id, label]) => (
                  <option key={id} value={id}>
                    {label}
                  </option>
                ))}
              </select>
              {bulkSport !== "" && (
                <select
                  value={bulkSubSport}
                  onChange={(e) => setBulkSubSport(e.target.value)}
                  className="cursor-pointer rounded border border-gray-200 px-1.5 py-0.5 text-xs text-gray-700 focus:outline-none focus:ring-1 focus:ring-gray-300"
                >
                  <option value="">Sub-sport…</option>
                  {(SUB_SPORTS_BY_SPORT[parseInt(bulkSport)] ?? []).map(
                    (id) => (
                      <option key={id} value={String(id)}>
                        {SUB_SPORT_LABELS[id]}
                      </option>
                    )
                  )}
                </select>
              )}
              <button
                onClick={bulkSetSport}
                disabled={!bulkSport || bulkEditMutation.isPending}
                className="cursor-pointer text-xs text-gray-500 hover:text-gray-700 disabled:opacity-50"
              >
                Set sport on all
              </button>
              <span className="text-gray-200">|</span>
              <div className="flex min-w-48 flex-1 items-center gap-2">
                <TagsInput value={bulkTags} onChange={setBulkTags} />
                <button
                  onClick={() => void bulkSetTags()}
                  disabled={bulkTags.length === 0 || bulkEditMutation.isPending}
                  className="shrink-0 cursor-pointer text-xs text-gray-500 hover:text-gray-700 disabled:opacity-50"
                >
                  Set tags on all
                </button>
              </div>
            </div>
          )}
          {editingLoading ? (
            <p className="text-sm text-gray-400">Loading…</p>
          ) : (
            <ul className="divide-y divide-gray-100 rounded-lg border border-gray-200 bg-white">
              {pendingTracks.map((track) => (
                <li
                  key={track.uuid}
                  className="flex items-center gap-3 px-4 py-2.5"
                >
                  <div className="h-10 w-10 shrink-0 overflow-hidden rounded bg-gray-50">
                    <img
                      src={`/api/tracks/${track.uuid}/preview.svg`}
                      alt=""
                      className="h-full w-full object-contain"
                    />
                  </div>
                  <div className="min-w-0 flex-1">
                    <Link
                      to={`/tracks/${track.uuid}`}
                      className="block truncate text-sm font-medium text-gray-900 hover:underline"
                    >
                      {track.name}
                    </Link>
                    <div className="mt-0.5 flex flex-wrap items-center gap-1.5">
                      <span className="text-xs text-gray-500">
                        {formatDistance(track.totalDistanceM)} ·{" "}
                        {formatAscent(track.totalAscentM)}
                      </span>
                      <span
                        className={`text-xs ${track.public ? "text-green-600" : "text-gray-400"}`}
                      >
                        {track.public ? "public" : "private"}
                      </span>
                      <span className="text-xs text-gray-500">
                        {SPORT_LABELS[track.sport] ?? track.sport}
                        {track.subSport !== 0 && (
                          <>
                            {" "}
                            (
                            {SUB_SPORT_LABELS[track.subSport] ?? track.subSport}
                            )
                          </>
                        )}
                      </span>
                      {track.tags.map((tag) => (
                        <span
                          key={tag}
                          className="rounded-full border border-gray-200 bg-gray-100 px-2 py-px text-xs text-gray-600"
                        >
                          {tag}
                        </span>
                      ))}
                    </div>
                  </div>
                  <button
                    onClick={() => dismissTrack(track.uuid)}
                    className="shrink-0 cursor-pointer text-xs text-gray-300 hover:text-gray-500"
                  >
                    Dismiss
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  )
}
