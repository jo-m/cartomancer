import { useEffect, useRef, useState } from "react"
import { useSearchParams, useNavigate, Link } from "react-router-dom"
import { useSession } from "../context/SessionContext"
import { fetchClient } from "../api/client"

export default function ConfirmEmail() {
  const [searchParams] = useSearchParams()
  const token = searchParams.get("token")
  const navigate = useNavigate()
  const { invalidateSession } = useSession()
  const [error, setError] = useState<string | null>(null)
  const [confirming, setConfirming] = useState(true)
  const submitted = useRef(false)

  useEffect(() => {
    if (!token || submitted.current) return
    submitted.current = true

    fetchClient
      .POST("/register/confirm", { body: { token } })
      .then(() => {
        invalidateSession()
        navigate("/")
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : "Confirmation failed")
        setConfirming(false)
      })
  }, [token, navigate, invalidateSession])

  if (!token) {
    return (
      <div className="flex min-h-[calc(100vh-57px)] items-center justify-center">
        <div className="w-full max-w-sm rounded-lg border border-gray-200 bg-white p-8 shadow-sm">
          <p className="text-sm text-red-600">Missing confirmation token.</p>
        </div>
      </div>
    )
  }

  if (confirming) {
    return (
      <div className="flex min-h-[calc(100vh-57px)] items-center justify-center">
        <p className="text-sm text-gray-600">Confirming your email…</p>
      </div>
    )
  }

  return (
    <div className="flex min-h-[calc(100vh-57px)] items-center justify-center">
      <div className="w-full max-w-sm rounded-lg border border-gray-200 bg-white p-8 shadow-sm">
        <h1 className="mb-4 text-xl font-semibold text-gray-900">
          Confirmation failed
        </h1>
        <p className="mb-4 text-sm text-red-600">{error}</p>
        <Link to="/login" className="text-sm text-gray-900 hover:underline">
          Go to login
        </Link>
      </div>
    </div>
  )
}
