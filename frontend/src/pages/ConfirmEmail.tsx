import { useState } from "react"
import { useSearchParams } from "react-router-dom"
import { useSession } from "../context/SessionContext"
import { fetchClient } from "../api/client"
import useDocumentTitle from "../hooks/useDocumentTitle"
import useToast from "../hooks/useToast"
import Alert from "../components/ui/Alert"
import Card from "../components/ui/Card"
import Button from "../components/ui/Button"
import Toast from "../components/Toast"

export default function ConfirmEmail() {
  useDocumentTitle("Confirm Email")
  const [searchParams] = useSearchParams()
  const token = searchParams.get("token")
  const { invalidateSession } = useSession()
  const { toast, showToast, dismissToast } = useToast()
  const [confirming, setConfirming] = useState(false)
  const [confirmed, setConfirmed] = useState(false)

  async function handleConfirm() {
    if (!token) return
    setConfirming(true)
    try {
      await fetchClient.POST("/confirm-email", { body: { token } })
      invalidateSession()
      setConfirmed(true)
    } catch (err) {
      showToast(err instanceof Error ? err.message : "Confirmation failed")
    } finally {
      setConfirming(false)
    }
  }

  if (!token) {
    return (
      <div className="flex min-h-[calc(100vh-var(--nav-height))] items-center justify-center px-4">
        <Card className="w-full max-w-sm p-8 shadow-sm">
          <Alert variant="error">Missing confirmation token.</Alert>
        </Card>
      </div>
    )
  }

  if (confirmed) {
    return (
      <div className="flex min-h-[calc(100vh-var(--nav-height))] items-center justify-center px-4">
        <Card className="w-full max-w-sm p-8 shadow-sm">
          <h1 className="mb-4 text-xl font-semibold text-text">
            Email confirmed
          </h1>
          <p className="mb-4 text-sm text-text-secondary">
            Your email has been confirmed. Please log in to continue.
          </p>
          <Button to="/login" variant="primary">
            Go to login
          </Button>
        </Card>
      </div>
    )
  }

  return (
    <div className="flex min-h-[calc(100vh-var(--nav-height))] items-center justify-center px-4">
      {toast && (
        <Toast
          key={toast.key}
          message={toast.message}
          variant={toast.variant}
          onDismiss={dismissToast}
        />
      )}
      <Card className="w-full max-w-sm p-8 shadow-sm">
        <h1 className="mb-4 text-xl font-semibold text-text">
          Confirm your email
        </h1>
        <p className="mb-6 text-sm text-text-secondary">
          Click the button below to confirm your email address.
        </p>
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
