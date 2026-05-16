import { useEffect } from "react"

const BASE_TITLE = "Cartomancer"

/**
 * Sets `document.title` based on `title`:
 * - non-empty string: `"<title> | Cartomancer"`
 * - empty string: `"Cartomancer"` alone (explicit bare title)
 * - `undefined`: no-op (used by dynamic pages while data is still loading,
 *   to avoid flickering the title from `"Cartomancer"` to `"<name> | Cartomancer"`)
 *
 * Resets to the base title on unmount.
 *
 * @param title - Page-specific title segment, `""` for the bare title, or
 *   `undefined` to skip the side-effect entirely (data not ready yet).
 */
export default function useDocumentTitle(title: string | undefined) {
  useEffect(() => {
    if (title === undefined) return
    document.title = title === "" ? BASE_TITLE : `${title} | ${BASE_TITLE}`
    return () => {
      document.title = BASE_TITLE
    }
  }, [title])
}
