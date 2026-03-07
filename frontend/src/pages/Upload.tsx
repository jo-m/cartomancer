import { useState, useRef, useEffect, useCallback } from "react"
import { fetchClient } from "../api/client"

type UploadStatus = "pending" | "uploading" | "done" | "error"

interface FileUpload {
  id: string
  file: File | null // null for items restored from session
  filename: string
  status: UploadStatus
  trackId?: string
  errorMsg?: string
}

const SESSION_KEY = "upload_history"

function loadFromSession(): FileUpload[] {
  try {
    const raw = sessionStorage.getItem(SESSION_KEY)
    if (!raw) return []
    return JSON.parse(raw) as FileUpload[]
  } catch {
    return []
  }
}

function persistToSession(uploads: FileUpload[]) {
  const settled = uploads
    .filter((u) => u.status === "done" || u.status === "error")
    .map(({ id, filename, status, trackId, errorMsg }) => ({
      id,
      filename,
      status,
      trackId,
      errorMsg,
      file: null,
    }))
  sessionStorage.setItem(SESSION_KEY, JSON.stringify(settled))
}

async function uploadFile(
  item: FileUpload,
  update: (id: string, patch: Partial<FileUpload>) => void
) {
  update(item.id, { status: "uploading" })
  try {
    const { data } = await fetchClient.POST("/tracks", {
      body: { file: item.file! } as never,
      bodySerializer(body) {
        const fd = new FormData()
        fd.append("file", item.file!)
        void body
        return fd
      },
    })
    update(item.id, { status: "done", trackId: data?.uuid })
  } catch (e) {
    update(item.id, { status: "error", errorMsg: (e as Error).message })
  }
}

export default function Upload() {
  const [uploads, setUploads] = useState<FileUpload[]>(() => loadFromSession())
  const [isDragging, setIsDragging] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    persistToSession(uploads)
  }, [uploads])

  const updateUpload = useCallback((id: string, patch: Partial<FileUpload>) => {
    setUploads((prev) =>
      prev.map((u) => (u.id === id ? { ...u, ...patch } : u))
    )
  }, [])

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
    newItems.forEach((item) => void uploadFile(item, updateUpload))
  }

  function dismiss(id: string) {
    setUploads((prev) => prev.filter((u) => u.id !== id))
  }

  function dismissAll() {
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

  const hasSettled = uploads.some(
    (u) => u.status === "done" || u.status === "error"
  )

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

      {uploads.length > 0 && (
        <div className="mt-4">
          {hasSettled && (
            <div className="mb-2 flex justify-end">
              <button
                onClick={dismissAll}
                className="cursor-pointer text-xs text-gray-400 hover:text-gray-600"
              >
                Dismiss all
              </button>
            </div>
          )}
          <ul className="divide-y divide-gray-100 rounded-lg border border-gray-200 bg-white">
            {uploads.map((u) => (
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
                {u.status === "done" && (
                  <span className="shrink-0 text-green-600">
                    Done
                    {u.trackId && (
                      <span className="ml-1 font-mono text-xs text-gray-400">
                        {u.trackId}
                      </span>
                    )}
                  </span>
                )}
                {u.status === "error" && (
                  <span className="shrink-0 text-red-600">{u.errorMsg}</span>
                )}
                {(u.status === "done" || u.status === "error") && (
                  <button
                    onClick={() => dismiss(u.id)}
                    className="shrink-0 cursor-pointer text-xs text-gray-300 hover:text-gray-500"
                  >
                    Dismiss
                  </button>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  )
}
