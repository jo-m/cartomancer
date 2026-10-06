import { useSyncExternalStore } from "react"

/** How often subscribed components re-read the clock. */
const TICK_MS = 60_000

function subscribe(onChange: () => void): () => void {
  const id = setInterval(onChange, TICK_MS)
  return () => clearInterval(id)
}

/** Bucketed to whole minutes so the snapshot stays stable between ticks. */
function getSnapshot(): number {
  return Math.floor(Date.now() / TICK_MS) * TICK_MS
}

/**
 * Returns the current time in epoch milliseconds, refreshed once per minute.
 * Rendering time-relative UI from a subscribed clock keeps components pure
 * and lets the UI advance on its own while the page stays open.
 */
export function useNowMs(): number {
  return useSyncExternalStore(subscribe, getSnapshot)
}
