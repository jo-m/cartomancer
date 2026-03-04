import type { paths } from "./schema.gen"

type AppConfig =
  paths["/app_config"]["get"]["responses"]["200"]["content"]["application/json"]
type SessionResponse =
  paths["/sessions"]["get"]["responses"]["200"]["content"]["application/json"]
type LoginRequest =
  paths["/sessions/login"]["post"]["requestBody"]["content"]["application/json"]
export type User =
  paths["/account"]["patch"]["responses"]["200"]["content"]["application/json"]
type UpdateMeRequest =
  paths["/account"]["patch"]["requestBody"]["content"]["application/json"]
type ChangePasswordRequest =
  paths["/account/change-password"]["post"]["requestBody"]["content"]["application/json"]

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init)
  if (!res.ok) {
    const body = await res.json().catch(() => ({ msg: res.statusText }))
    throw new Error(body.msg || res.statusText)
  }
  return res.json() as Promise<T>
}

async function fetchEmpty(url: string, init?: RequestInit): Promise<void> {
  const res = await fetch(url, init)
  if (!res.ok) {
    const body = await res.json().catch(() => ({ msg: res.statusText }))
    throw new Error(body.msg || res.statusText)
  }
}

export function getAppConfig(): Promise<AppConfig> {
  return fetchJSON<AppConfig>("/api/app_config")
}

export function getSession(): Promise<SessionResponse> {
  return fetchJSON<SessionResponse>("/api/sessions")
}

export function login(req: LoginRequest): Promise<SessionResponse> {
  return fetchJSON<SessionResponse>("/api/sessions/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  })
}

export function logout(): Promise<void> {
  return fetchEmpty("/api/sessions/logout", { method: "POST" })
}

export function updateMe(req: UpdateMeRequest): Promise<User> {
  return fetchJSON<User>("/api/account", {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  })
}

export function changePassword(req: ChangePasswordRequest): Promise<void> {
  return fetchEmpty("/api/account/change-password", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  })
}

export function deleteMe(): Promise<void> {
  return fetchEmpty("/api/account", { method: "DELETE" })
}
