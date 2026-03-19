import { Link } from "react-router-dom"
import { $api } from "../api/client"

/** Welcome landing page shown to all visitors at the root route. */
export default function Welcome() {
  const { data: appConfig } = $api.useQuery("get", "/app_config")

  return (
    <div className="mx-auto max-w-5xl px-4 py-16">
      <h1 className="mb-4 text-3xl font-semibold text-gray-900">Detour</h1>
      <p className="mb-8 text-gray-600">
        Share and explore GPX tracks for cycling and running.
      </p>
      <div className="flex gap-4">
        <Link
          to="/tracks"
          className="rounded bg-gray-900 px-4 py-2 text-sm text-white hover:bg-gray-700"
        >
          Browse tracks
        </Link>
        {appConfig?.registrationEnabled && (
          <Link
            to="/register"
            className="rounded border border-gray-300 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
          >
            Create account
          </Link>
        )}
      </div>
    </div>
  )
}
