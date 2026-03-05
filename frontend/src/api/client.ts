import createFetchClient from "openapi-fetch"
import createClient from "openapi-react-query"
import type { QueryClient } from "@tanstack/react-query"
import type { paths } from "./schema.gen"

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
      throw new Error(`Error: ${msg}`)
    }
    return response
  },
})

export const $api = createClient(fetchClient)

export type User =
  paths["/account"]["patch"]["responses"]["200"]["content"]["application/json"]
