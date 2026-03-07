import { useState, useRef } from "react"
import { fetchClient } from "../api/client"

type UploadStatus = "pending" | "uploading" | "done" | "error"

interface FileUpload {
  id: string
  file: File
  status: UploadStatus
  errorMsg?: string
}

async function uploadFile(
  item: FileUpload,
  update: (id: string, patch: Partial<FileUpload>) => void
) {
  update(item.id, { status: "uploading" })
  try {
    await fetchClient.POST("/tracks", {
      body: { file: item.file } as never,
      bodySerializer(body) {
        const fd = new FormData()
        fd.append("file", item.file)
        void body
        return fd
      },
    })
    update(item.id, { status: "done" })
  } catch (e) {
    update(item.id, { status: "error", errorMsg: (e as Error).message })
  }
}

export default function Upload() {
  const [uploads, setUploads] = useState<FileUpload[]>([])
  const [isDragging, setIsDragging] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  function updateUpload(id: string, patch: Partial<FileUpload>) {
    setUploads((prev) =>
      prev.map((u) => (u.id === id ? { ...u, ...patch } : u))
    )
  }

  function addFiles(files: FileList | File[]) {
    const valid = Array.from(files).filter(
      (f) => f.name.endsWith(".gpx") || f.name.endsWith(".fit")
    )
    if (valid.length === 0) return
    const newItems: FileUpload[] = valid.map((f) => ({
      id: crypto.randomUUID(),
      file: f,
      status: "pending",
    }))
    setUploads((prev) => [...prev, ...newItems])
    newItems.forEach((item) => void uploadFile(item, updateUpload))
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
        <ul className="mt-4 divide-y divide-gray-100 rounded-lg border border-gray-200 bg-white">
          {uploads.map((u) => (
            <li
              key={u.id}
              className="flex items-center gap-3 px-4 py-2.5 text-sm"
            >
              <span className="min-w-0 flex-1 truncate text-gray-800">
                {u.file.name}
              </span>
              {u.status === "pending" && (
                <span className="shrink-0 text-gray-400">Pending</span>
              )}
              {u.status === "uploading" && (
                <span className="shrink-0 text-blue-500">Uploading…</span>
              )}
              {u.status === "done" && (
                <span className="shrink-0 text-green-600">Done</span>
              )}
              {u.status === "error" && (
                <span className="shrink-0 text-red-600">
                  {u.errorMsg}
                </span>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
