import createFetchClient from "openapi-fetch"
import createClient from "openapi-react-query"
import type { QueryClient } from "@tanstack/react-query"
import type { paths } from "./schema.gen"

/** An error thrown for non-2xx API responses, carrying the HTTP status code. */
export class ApiError extends Error {
  readonly status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = "ApiError"
    this.status = status
  }
}

declare module "@tanstack/react-query" {
  interface Register {
    defaultError: Error
  }
}

export const fetchClient = createFetchClient<paths>({ baseUrl: "/api" })

let queryClient: QueryClient | null = null

export function setQueryClient(qc: QueryClient) {
  queryClient = qc
}

fetchClient.use({
  async onRequest({ request }) {
    request.headers.set("X-Requested-With", "cartomancer")
    return request
  },
  async onResponse({ request, response }) {
    if (!response.ok) {
      if (
        response.status === 401 &&
        !request.url.endsWith("/sessions") &&
        !request.url.endsWith("/sessions/login")
      ) {
        queryClient?.setQueryData(["get", "/sessions"], { user: null })
        window.location.assign("/login")
        return response
      }
      const body = await response
        .clone()
        .json()
        .catch(() => null)
      const msg = (body as { msg?: string } | null)?.msg ?? response.statusText
      throw new ApiError(`Error: ${msg}`, response.status)
    }
    return response
  },
})

export const $api = createClient(fetchClient)

/** Shared hook for fetching the app configuration. */
export function useAppConfig() {
  return $api.useQuery("get", "/app_config")
}

export type User =
  paths["/account"]["patch"]["responses"]["200"]["content"]["application/json"]
