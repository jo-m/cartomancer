import { useEffect } from "react"

const BASE_TITLE = "Cartomancer"

/**
 * Sets `document.title` to `title | Cartomancer`, or just `Cartomancer` if no title is given.
 * Resets to the base title on unmount.
 *
 * @param title - Page-specific title segment, or undefined to use the base title alone.
 */
export default function useDocumentTitle(title?: string) {
  useEffect(() => {
    document.title = title ? `${title} | ${BASE_TITLE}` : BASE_TITLE
    return () => {
      document.title = BASE_TITLE
    }
  }, [title])
}
