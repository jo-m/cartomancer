import { useState, useEffect } from "react"

interface SvgPreviewProps {
  src: string
  alt: string
  className?: string
}

/**
 * prepareSvg strips fixed width/height attributes from an SVG string and adds
 * a viewBox (derived from the original dimensions) so the SVG scales to fill
 * its container. Preserves any existing viewBox.
 */
function prepareSvg(raw: string): string {
  const parser = new DOMParser()
  const doc = parser.parseFromString(raw, "image/svg+xml")
  const svg = doc.querySelector("svg")
  if (!svg) return raw

  const w = svg.getAttribute("width")
  const h = svg.getAttribute("height")
  if (w && h && !svg.getAttribute("viewBox")) {
    svg.setAttribute("viewBox", `0 0 ${w} ${h}`)
  }
  svg.removeAttribute("width")
  svg.removeAttribute("height")
  svg.style.width = "100%"
  svg.style.height = "100%"

  return svg.outerHTML
}

/**
 * SvgPreview fetches an SVG from a URL and renders it inline so that
 * `currentColor` in the SVG inherits from the parent's CSS `color`.
 * Fixed width/height attributes are replaced with a viewBox so the
 * SVG scales responsively within its container.
 */
export default function SvgPreview({ src, alt, className }: SvgPreviewProps) {
  const [svg, setSvg] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    fetch(src)
      .then((r) => (r.ok ? r.text() : Promise.reject(r.statusText)))
      .then((text) => {
        if (!cancelled) setSvg(prepareSvg(text))
      })
      .catch(() => {
        if (!cancelled) setSvg(null)
      })
    return () => {
      cancelled = true
    }
  }, [src])

  if (!svg) {
    return <div className={className} role="img" aria-label={alt} />
  }

  return (
    <div
      className={className}
      role="img"
      aria-label={alt}
      dangerouslySetInnerHTML={{ __html: svg }}
    />
  )
}
