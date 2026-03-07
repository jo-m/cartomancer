import { createContext, useContext, useCallback } from "react"
import type { ReactNode } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { $api, fetchClient } from "../api/client"
import type { User } from "../api/client"

interface SessionState {
  user: User | null
  loading: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
  invalidateSession: () => Promise<void>
}

const SessionContext = createContext<SessionState | null>(null)

export function SessionProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient()

  const { data, isLoading } = $api.useQuery("get", "/sessions")
  const user = data?.user ?? null

  const invalidateSession = useCallback(
    () => queryClient.invalidateQueries({ queryKey: ["get", "/sessions"] }),
    [queryClient]
  )

  const login = useCallback(
    async (email: string, password: string) => {
      const { data: loginData } = await fetchClient.POST("/sessions/login", {
        body: { email, password },
      })
      queryClient.setQueryData(["get", "/sessions"], {
        user: loginData?.user ?? null,
      })
    },
    [queryClient]
  )

  const logout = useCallback(async () => {
    await fetchClient.POST("/sessions/logout")
    sessionStorage.clear()
    queryClient.setQueryData(["get", "/sessions"], { user: null })
  }, [queryClient])

  return (
    <SessionContext.Provider
      value={{ user, loading: isLoading, login, logout, invalidateSession }}
    >
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
