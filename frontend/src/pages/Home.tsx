import { useEffect, useState } from "react"
import { getAppConfig } from "../api/client"

interface AppConfig {
  appName: string
  externalBaseUrl: string
}

export default function Home() {
  const [config, setConfig] = useState<AppConfig | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    getAppConfig()
      .then(setConfig)
      .catch((err: unknown) => {
        setError(err instanceof Error ? err.message : "Unknown error")
      })
  }, [])

  if (error) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-50">
        <div className="rounded-lg border border-red-200 bg-red-50 p-6">
          <p className="text-red-800">Failed to load config: {error}</p>
        </div>
      </div>
    )
  }

  if (!config) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-50">
        <p className="text-gray-500">Loading...</p>
      </div>
    )
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50">
      <div className="rounded-lg border border-gray-200 bg-white p-8 shadow-sm">
        <h1 className="mb-4 text-2xl font-bold text-gray-900">
          {config.appName}
        </h1>
        <dl className="space-y-2 text-sm">
          <div>
            <dt className="font-medium text-gray-500">App Name</dt>
            <dd className="text-gray-900">{config.appName}</dd>
          </div>
          <div>
            <dt className="font-medium text-gray-500">External Base URL</dt>
            <dd className="text-gray-900">{config.externalBaseUrl}</dd>
          </div>
        </dl>
      </div>
    </div>
  )
}
