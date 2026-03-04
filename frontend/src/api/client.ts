import createFetchClient from "openapi-fetch"
import createClient from "openapi-react-query"
import type { paths } from "./schema.gen"

declare module "@tanstack/react-query" {
  interface Register {
    defaultError: Error
  }
}

export const fetchClient = createFetchClient<paths>({ baseUrl: "/api" })

fetchClient.use({
  async onResponse({ response }) {
    if (!response.ok) {
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
