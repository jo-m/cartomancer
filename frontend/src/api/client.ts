import type { paths } from "./schema.gen"

type AppConfig =
  paths["/app_config"]["get"]["responses"]["200"]["content"]["application/json"]

async function fetchJSON<T>(url: string): Promise<T> {
  const res = await fetch(url)
  if (!res.ok) {
    const body = await res.json().catch(() => ({ msg: res.statusText }))
    throw new Error(body.msg || res.statusText)
  }
  return res.json() as Promise<T>
}

export function getAppConfig(): Promise<AppConfig> {
  return fetchJSON<AppConfig>("/api/app_config")
}
