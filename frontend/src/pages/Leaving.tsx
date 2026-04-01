import { useSearchParams, Link, useNavigate } from "react-router-dom"
import { useAppConfig } from "../api/client"
import Button from "../components/ui/Button"

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
        <h1 className="text-xl font-semibold text-text">Invalid link</h1>
        <p className="mt-2 text-sm text-text-secondary">
          No destination URL provided.
        </p>
        <Link
          to="/"
          className="mt-4 inline-block text-sm text-text-secondary underline hover:text-text transition-colors"
        >
          Go back home
        </Link>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-lg px-4 py-16 text-center">
      <h1 className="text-xl font-semibold text-text">
        You are leaving {name}
      </h1>
      <p className="mt-3 text-sm text-text-secondary">
        You are about to visit an external link. This link has not been verified
        and may lead to a third-party site.
      </p>
      <p className="mt-4 break-all rounded bg-panel border border-border px-4 py-3 text-sm font-mono text-text-secondary">
        {target}
      </p>
      <div className="mt-6 flex items-center justify-center gap-4">
        <Button variant="secondary" onClick={() => navigate(-1)}>
          Go back
        </Button>
        <a href={target} rel="noopener noreferrer">
          <Button variant="primary">Continue to site</Button>
        </a>
      </div>
    </div>
  )
}
