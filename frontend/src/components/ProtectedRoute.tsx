import { Navigate } from "react-router-dom"
import { useSession } from "../context/SessionContext"

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

/** Route wrapper that only allows access to admin users. Redirects to home otherwise. */
export function AdminRoute({ children }: { children: React.ReactNode }) {
  const { user, loading } = useSession()
  if (loading) return null
  if (!user) return <Navigate to="/login" replace />
  if (!user.admin) return <Navigate to="/" replace />
  return children
}
