import { Navigate } from "react-router-dom"
import { useSession } from "../context/SessionContext"
import Forbidden from "../pages/Forbidden"

export function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { user, loading } = useSession()
  if (loading) return null
  if (!user) return <Navigate to="/login" replace />
  return children
}

export function GuestRoute({ children }: { children: React.ReactNode }) {
  const { user, loading } = useSession()
  if (loading) return null
  if (user) return <Navigate to="/" replace />
  return children
}

/**
 * Route wrapper that only allows access to admin users.
 * Unauthenticated users are sent to the login page; authenticated non-admin
 * users see a 403 Forbidden page so the denial is visible rather than silent.
 */
export function AdminRoute({ children }: { children: React.ReactNode }) {
  const { user, loading } = useSession()
  if (loading) return null
  if (!user) return <Navigate to="/login" replace />
  if (!user.admin) return <Forbidden />
  return children
}
