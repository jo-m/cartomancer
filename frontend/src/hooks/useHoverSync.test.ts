// @vitest-environment jsdom
import { describe, it, expect, vi } from "vitest"
import { act, renderHook } from "@testing-library/react"
import { createHoverStore, useHoverStore, useHoverValue } from "./useHoverSync"

describe("createHoverStore", () => {
  it("starts with a null value", () => {
    const store = createHoverStore()
    expect(store.get()).toBeNull()
  })

  it("set updates the value and notifies subscribers", () => {
    const store = createHoverStore()
    const listener = vi.fn()
    store.subscribe(listener)
    store.set(3)
    expect(store.get()).toBe(3)
    expect(listener).toHaveBeenCalledTimes(1)
  })

  it("set with the same value is a no-op (no notification)", () => {
    const store = createHoverStore()
    const listener = vi.fn()
    store.subscribe(listener)
    store.set(7)
    store.set(7)
    expect(listener).toHaveBeenCalledTimes(1)
  })

  it("subscribe returns an unsubscribe function", () => {
    const store = createHoverStore()
    const listener = vi.fn()
    const unsubscribe = store.subscribe(listener)
    store.set(1)
    unsubscribe()
    store.set(2)
    expect(listener).toHaveBeenCalledTimes(1)
    expect(store.get()).toBe(2)
  })

  it("notifies multiple independent subscribers", () => {
    const store = createHoverStore()
    const a = vi.fn()
    const b = vi.fn()
    store.subscribe(a)
    store.subscribe(b)
    store.set(42)
    expect(a).toHaveBeenCalledTimes(1)
    expect(b).toHaveBeenCalledTimes(1)
  })

  it("supports resetting to null", () => {
    const store = createHoverStore()
    store.set(5)
    store.set(null)
    expect(store.get()).toBeNull()
  })
})

describe("useHoverStore", () => {
  it("returns a stable store across renders", () => {
    const { result, rerender } = renderHook(() => useHoverStore())
    const first = result.current
    rerender()
    expect(result.current).toBe(first)
  })
})

describe("useHoverValue", () => {
  it("returns the current store value", () => {
    const store = createHoverStore()
    store.set(2)
    const { result } = renderHook(() => useHoverValue(store))
    expect(result.current).toBe(2)
  })

  it("updates when the store changes", () => {
    const store = createHoverStore()
    const { result } = renderHook(() => useHoverValue(store))
    expect(result.current).toBeNull()
    act(() => store.set(9))
    expect(result.current).toBe(9)
    act(() => store.set(null))
    expect(result.current).toBeNull()
  })
})
