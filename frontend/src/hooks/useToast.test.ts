// @vitest-environment jsdom
import { describe, it, expect } from "vitest"
import { act, renderHook } from "@testing-library/react"
import useToast from "./useToast"

describe("useToast", () => {
  it("starts with no toast", () => {
    const { result } = renderHook(() => useToast())
    expect(result.current.toast).toBeNull()
  })

  it("defaults to the error variant", () => {
    const { result } = renderHook(() => useToast())
    act(() => result.current.showToast("oops"))
    expect(result.current.toast).toMatchObject({
      message: "oops",
      variant: "error",
    })
  })

  it("supports the success variant", () => {
    const { result } = renderHook(() => useToast())
    act(() => result.current.showToast("yay", "success"))
    expect(result.current.toast).toMatchObject({
      message: "yay",
      variant: "success",
    })
  })

  it("increments the key when re-showing so identical messages re-trigger", () => {
    const { result } = renderHook(() => useToast())
    act(() => result.current.showToast("same"))
    const firstKey = result.current.toast!.key
    act(() => result.current.showToast("same"))
    expect(result.current.toast!.key).toBe(firstKey + 1)
    act(() => result.current.showToast("same"))
    expect(result.current.toast!.key).toBe(firstKey + 2)
  })

  it("dismiss clears the toast", () => {
    const { result } = renderHook(() => useToast())
    act(() => result.current.showToast("temp"))
    expect(result.current.toast).not.toBeNull()
    act(() => result.current.dismissToast())
    expect(result.current.toast).toBeNull()
  })

  it("exposes stable callback references across renders", () => {
    const { result, rerender } = renderHook(() => useToast())
    const { showToast, dismissToast } = result.current
    rerender()
    expect(result.current.showToast).toBe(showToast)
    expect(result.current.dismissToast).toBe(dismissToast)
  })
})
