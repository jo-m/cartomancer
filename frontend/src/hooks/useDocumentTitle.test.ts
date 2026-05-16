// @vitest-environment jsdom
import { describe, it, expect, beforeEach } from "vitest"
import { renderHook } from "@testing-library/react"
import useDocumentTitle from "./useDocumentTitle"

describe("useDocumentTitle", () => {
  beforeEach(() => {
    document.title = "Cartomancer"
  })

  it("sets the document title with the base suffix", () => {
    renderHook(() => useDocumentTitle("My Page"))
    expect(document.title).toBe("My Page | Cartomancer")
  })

  it("sets the bare base title for an empty string", () => {
    renderHook(() => useDocumentTitle(""))
    expect(document.title).toBe("Cartomancer")
  })

  it("does nothing when title is undefined", () => {
    document.title = "preset by caller"
    renderHook(() => useDocumentTitle(undefined))
    expect(document.title).toBe("preset by caller")
  })

  it("updates the title when the argument changes", () => {
    const { rerender } = renderHook(({ t }) => useDocumentTitle(t), {
      initialProps: { t: "First" as string | undefined },
    })
    expect(document.title).toBe("First | Cartomancer")
    rerender({ t: "Second" })
    expect(document.title).toBe("Second | Cartomancer")
  })

  it("resets to the base title on unmount", () => {
    const { unmount } = renderHook(() => useDocumentTitle("Temp"))
    expect(document.title).toBe("Temp | Cartomancer")
    unmount()
    expect(document.title).toBe("Cartomancer")
  })
})
