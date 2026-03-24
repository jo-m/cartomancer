import { useSearchParams, Link, useNavigate } from "react-router-dom"
import { useAppConfig } from "../api/client"

/** Interstitial page warning users they are about to leave the app via an external link. */
export default function Leaving() {
  const [params] = useSearchParams()
  const navigate = useNavigate()
  const { data: appConfig } = useAppConfig()
  const target = params.get("url")
  const name = appConfig?.instanceName ?? "this site"

  if (!target) {
    return (
      <div className="mx-auto max-w-lg px-4 py-16 text-center">
        <h1 className="text-xl font-semibold text-gray-900">Invalid link</h1>
        <p className="mt-2 text-sm text-gray-600">
          No destination URL provided.
        </p>
        <Link
          to="/"
          className="mt-4 inline-block text-sm text-gray-700 underline hover:text-gray-900"
        >
          Go back home
        </Link>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-lg px-4 py-16 text-center">
      <h1 className="text-xl font-semibold text-gray-900">
        You are leaving {name}
      </h1>
      <p className="mt-3 text-sm text-gray-600">
        You are about to visit an external link. This link has not been verified
        and may lead to a third-party site.
      </p>
      <p className="mt-4 break-all rounded bg-gray-50 px-4 py-3 text-sm font-mono text-gray-700">
        {target}
      </p>
      <div className="mt-6 flex items-center justify-center gap-4">
        <button
          onClick={() => navigate(-1)}
          className="rounded border border-gray-300 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
        >
          Go back
        </button>
        <a
          href={target}
          rel="noopener noreferrer"
          className="rounded bg-gray-900 px-4 py-2 text-sm text-white hover:bg-gray-700"
        >
          Continue to site
        </a>
      </div>
    </div>
  )
}
