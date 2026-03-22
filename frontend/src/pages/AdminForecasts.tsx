import { useState } from "react"
import { Link } from "react-router-dom"
import { $api } from "../api/client"

/** Formats a byte count into a human-readable size string. */
function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export default function AdminForecasts() {
  const [expandedId, setExpandedId] = useState<number | null>(null)
  const [search, setSearch] = useState("")

  const { data } = $api.useQuery("get", "/admin/forecasts", {})
  const forecasts = data?.forecasts ?? []

  const filtered = forecasts.filter(
    (f) =>
      f.attribution.text.toLowerCase().includes(search.toLowerCase()) ||
      f.referenceTime.includes(search)
  )

  return (
    <div className="mx-auto max-w-4xl px-4 py-8">
      <div className="mb-6 flex items-center gap-4">
        <h1 className="text-2xl font-semibold text-gray-900">Admin</h1>
        <Link
          to="/admin/users"
          className="pb-0.5 text-sm font-medium text-gray-500 hover:text-gray-700"
        >
          Users
        </Link>
        <Link
          to="/admin/forecasts"
          className="border-b-2 border-gray-900 pb-0.5 text-sm font-medium text-gray-900"
        >
          Forecasts
        </Link>
      </div>

      <div className="mb-4">
        <input
          type="text"
          placeholder="Search forecasts..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="w-full max-w-xs rounded border border-gray-300 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none"
        />
      </div>

      <div className="rounded-lg border border-gray-200 bg-white">
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-gray-200 text-xs font-medium text-gray-500">
              <th className="px-4 py-3">Reference time</th>
              <th className="px-4 py-3">Attribution</th>
              <th className="px-4 py-3">Bounds</th>
              <th className="px-4 py-3">Files</th>
              <th className="px-4 py-3">Created</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((f) => {
              const totalSize = f.files.reduce(
                (sum, file) => sum + file.fileSize,
                0
              )
              const isExpanded = expandedId === f.id

              return (
                <tr
                  key={f.id}
                  className="border-b border-gray-100 last:border-0"
                >
                  <td colSpan={5} className="px-0 py-0">
                    <button
                      onClick={() => setExpandedId(isExpanded ? null : f.id)}
                      className="flex w-full cursor-pointer items-center text-left"
                    >
                      <span className="w-1/5 px-4 py-3 text-gray-900">
                        {f.referenceTime.slice(0, 16).replace("T", " ")}
                      </span>
                      <span className="w-1/5 px-4 py-3 text-gray-600">
                        <a
                          href={f.attribution.href}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="hover:underline"
                          onClick={(e) => e.stopPropagation()}
                        >
                          {f.attribution.text}
                        </a>
                      </span>
                      <span className="w-1/5 px-4 py-3 text-xs text-gray-500">
                        {f.bounds
                          ? `${f.bounds.min.lat.toFixed(1)}, ${f.bounds.min.lon.toFixed(1)} - ${f.bounds.max.lat.toFixed(1)}, ${f.bounds.max.lon.toFixed(1)}`
                          : "--"}
                      </span>
                      <span className="w-1/5 px-4 py-3 text-gray-600">
                        {f.files.length} ({formatBytes(totalSize)})
                      </span>
                      <span className="w-1/5 px-4 py-3 text-gray-500">
                        {f.createdAt.slice(0, 10)}
                      </span>
                    </button>

                    {isExpanded && f.files.length > 0 && (
                      <div className="border-t border-gray-100 bg-gray-50 px-4 py-3">
                        <table className="w-full text-xs">
                          <thead>
                            <tr className="text-gray-500">
                              <th className="pb-1 pr-4 text-left font-medium">
                                Variable
                              </th>
                              <th className="pb-1 pr-4 text-left font-medium">
                                Valid time
                              </th>
                              <th className="pb-1 pr-4 text-left font-medium">
                                Valid until
                              </th>
                              <th className="pb-1 text-left font-medium">
                                Size
                              </th>
                            </tr>
                          </thead>
                          <tbody>
                            {f.files.map((file) => (
                              <tr key={file.id}>
                                <td className="py-0.5 pr-4 text-gray-700">
                                  {file.variable}
                                </td>
                                <td className="py-0.5 pr-4 text-gray-600">
                                  {file.validTime
                                    .slice(0, 16)
                                    .replace("T", " ")}
                                </td>
                                <td className="py-0.5 pr-4 text-gray-600">
                                  {file.validUntilTime
                                    .slice(0, 16)
                                    .replace("T", " ")}
                                </td>
                                <td className="py-0.5 text-gray-600">
                                  {formatBytes(file.fileSize)}
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    )}
                  </td>
                </tr>
              )
            })}
            {filtered.length === 0 && (
              <tr>
                <td
                  colSpan={5}
                  className="px-4 py-6 text-center text-sm text-gray-500"
                >
                  No forecasts found.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
