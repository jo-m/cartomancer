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
