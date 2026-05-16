import { describe, it, expect } from "vitest"
import { externalUrl } from "./externalUrl"

describe("externalUrl", () => {
  it("wraps a plain URL through the /leaving interstitial", () => {
    expect(externalUrl("https://example.com")).toBe(
      "/leaving?url=https%3A%2F%2Fexample.com"
    )
  })

  it("encodes query strings and special characters", () => {
    expect(externalUrl("https://example.com/a?b=c&d=e f")).toBe(
      "/leaving?url=https%3A%2F%2Fexample.com%2Fa%3Fb%3Dc%26d%3De%20f"
    )
  })
})
