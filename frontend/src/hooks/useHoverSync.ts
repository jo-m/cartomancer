import { useState, useSyncExternalStore } from "react"

/** A minimal external store for sharing a hover index across components without re-rendering their common parent. */
export interface HoverStore {
  /** Returns the current hover index. */
  get: () => number | null
  /** Sets the hover index and notifies subscribers. */
  set: (index: number | null) => void
  /** Subscribes to changes; returns an unsubscribe function. */
  subscribe: (cb: () => void) => () => void
}

/** Creates a HoverStore instance. */
export function createHoverStore(): HoverStore {
  let value: number | null = null
  const listeners = new Set<() => void>()
  return {
    get: () => value,
    set: (index) => {
      if (value === index) return
      value = index
      listeners.forEach((fn) => fn())
    },
    subscribe: (cb) => {
      listeners.add(cb)
      return () => listeners.delete(cb)
    },
  }
}

/** Creates a stable HoverStore that persists across renders. */
export function useHoverStore(): HoverStore {
  const [store] = useState(createHoverStore)
  return store
}

/** Subscribes to a HoverStore and returns the current hover index. */
export function useHoverValue(store: HoverStore): number | null {
  return useSyncExternalStore(store.subscribe, store.get)
}
