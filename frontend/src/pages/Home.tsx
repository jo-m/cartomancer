import { $api } from "../api/client"

export default function Home() {
  const { data, error, isLoading } = $api.useQuery("get", "/app_config")

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-50">
        <p className="text-gray-500">Loading...</p>
      </div>
    )
  }

  if (error || !data) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-50">
        <div className="rounded-lg border border-red-200 bg-red-50 p-6">
          <p className="text-red-800">
            Failed to load config:{" "}
            {(error as Error | null)?.message ?? "Unknown error"}
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50">
      <div className="rounded-lg border border-gray-200 bg-white p-8 shadow-sm">
        <h1 className="mb-4 text-2xl font-bold text-gray-900">
          {data.appName}
        </h1>
        <dl className="space-y-2 text-sm">
          <div>
            <dt className="font-medium text-gray-500">App Name</dt>
            <dd className="text-gray-900">{data.appName}</dd>
          </div>
          <div>
            <dt className="font-medium text-gray-500">External Base URL</dt>
            <dd className="text-gray-900">{data.externalBaseUrl}</dd>
          </div>
        </dl>
      </div>
    </div>
  )
}
