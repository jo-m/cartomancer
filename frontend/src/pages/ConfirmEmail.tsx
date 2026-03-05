import { useState } from "react"
import { useSearchParams, Link } from "react-router-dom"
import { useSession } from "../context/SessionContext"
import { fetchClient } from "../api/client"

export default function ConfirmEmail() {
  const [searchParams] = useSearchParams()
  const token = searchParams.get("token")
  const { invalidateSession } = useSession()
  const [error, setError] = useState<string | null>(null)
  const [confirming, setConfirming] = useState(false)
  const [confirmed, setConfirmed] = useState(false)

  async function handleConfirm() {
    if (!token) return
    setConfirming(true)
    setError(null)
    try {
      await fetchClient.POST("/register/confirm", { body: { token } })
      invalidateSession()
      setConfirmed(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Confirmation failed")
    } finally {
      setConfirming(false)
    }
  }

  if (!token) {
    return (
      <div className="flex min-h-[calc(100vh-57px)] items-center justify-center">
        <div className="w-full max-w-sm rounded-lg border border-gray-200 bg-white p-8 shadow-sm">
          <p className="text-sm text-red-600">Missing confirmation token.</p>
        </div>
      </div>
    )
  }

  if (confirmed) {
    return (
      <div className="flex min-h-[calc(100vh-57px)] items-center justify-center">
        <div className="w-full max-w-sm rounded-lg border border-gray-200 bg-white p-8 shadow-sm">
          <h1 className="mb-4 text-xl font-semibold text-gray-900">
            Email confirmed
          </h1>
          <p className="mb-4 text-sm text-gray-600">
            Your email has been confirmed. Please log in to continue.
          </p>
          <Link
            to="/login"
            className="inline-block rounded-md bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-800"
          >
            Go to login
          </Link>
        </div>
      </div>
    )
  }

  return (
    <div className="flex min-h-[calc(100vh-57px)] items-center justify-center">
      <div className="w-full max-w-sm rounded-lg border border-gray-200 bg-white p-8 shadow-sm">
        <h1 className="mb-4 text-xl font-semibold text-gray-900">
          Confirm your email
        </h1>
        <p className="mb-6 text-sm text-gray-600">
          Click the button below to confirm your email address.
        </p>
        {error && <p className="mb-4 text-sm text-red-600">{error}</p>}
        <button
          onClick={handleConfirm}
          disabled={confirming}
          className="w-full cursor-pointer rounded-md bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-800 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {confirming ? "Confirming…" : "Confirm email"}
        </button>
      </div>
    </div>
  )
}
