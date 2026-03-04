import { createContext, useContext, useEffect, useState } from "react"
import type { ReactNode } from "react"
import type { User } from "../api/client"
import {
  getSession,
  login as apiLogin,
  logout as apiLogout,
} from "../api/client"

interface SessionState {
  user: User | null
  loading: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
  setUser: (user: User | null) => void
}

const SessionContext = createContext<SessionState | null>(null)

export function SessionProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    getSession()
      .then((s) => setUser(s.user ?? null))
      .catch(() => setUser(null))
      .finally(() => setLoading(false))
  }, [])

  async function login(email: string, password: string) {
    const s = await apiLogin({ email, password })
    setUser(s.user ?? null)
  }

  async function logout() {
    await apiLogout()
    setUser(null)
  }

  return (
    <SessionContext.Provider value={{ user, loading, login, logout, setUser }}>
      {children}
    </SessionContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function useSession(): SessionState {
  const ctx = useContext(SessionContext)
  if (!ctx) throw new Error("useSession must be used within SessionProvider")
  return ctx
}
