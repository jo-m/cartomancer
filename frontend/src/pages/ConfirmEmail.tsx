import { useState } from "react"
import { useSearchParams, Link } from "react-router-dom"
import { useSession } from "../context/SessionContext"
import { fetchClient } from "../api/client"
import Card from "../components/ui/Card"
import Button from "../components/ui/Button"

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
      await fetchClient.POST("/confirm-email", { body: { token } })
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
      <div className="flex min-h-[calc(100vh-57px)] items-center justify-center px-4">
        <Card className="w-full max-w-sm p-8 shadow-sm">
          <p role="alert" className="text-sm text-error">
            Missing confirmation token.
          </p>
        </Card>
      </div>
    )
  }

  if (confirmed) {
    return (
      <div className="flex min-h-[calc(100vh-57px)] items-center justify-center px-4">
        <Card className="w-full max-w-sm p-8 shadow-sm">
          <h1 className="mb-4 text-xl font-semibold text-text">
            Email confirmed
          </h1>
          <p className="mb-4 text-sm text-text-secondary">
            Your email has been confirmed. Please log in to continue.
          </p>
          <Link to="/login">
            <Button variant="primary">Go to login</Button>
          </Link>
        </Card>
      </div>
    )
  }

  return (
    <div className="flex min-h-[calc(100vh-57px)] items-center justify-center px-4">
      <Card className="w-full max-w-sm p-8 shadow-sm">
        <h1 className="mb-4 text-xl font-semibold text-text">
          Confirm your email
        </h1>
        <p className="mb-6 text-sm text-text-secondary">
          Click the button below to confirm your email address.
        </p>
        {error && (
          <p role="alert" className="mb-4 text-sm text-error">
            {error}
          </p>
        )}
        <Button
          variant="primary"
          onClick={handleConfirm}
          disabled={confirming}
          className="w-full"
        >
          {confirming ? "Confirming..." : "Confirm email"}
        </Button>
      </Card>
    </div>
  )
}
