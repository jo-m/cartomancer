/**
 * Pure canvas-drawing helpers for the map's endpoint icon and direction
 * arrows. No React or OpenLayers state — the produced canvases are wrapped
 * into OL `Icon`s by the consumer.
 */

import { TRACK_LINE_HALO_WIDTH, TRACK_LINE_INNER_WIDTH } from "./trackMapStyle"

/** Target screen-space spacing between direction arrows, in CSS pixels. */
export const ARROW_SCREEN_SPACING_PX = 280

/** Logical pixel size of the square chevron icon (in CSS pixels). */
export const ARROW_CANVAS_SIZE = 18

/**
 * Chevron stroke widths relative to the track line. Drawing the chevron a bit
 * thinner than the track keeps the arrows visually subordinate to the route.
 */
export const ARROW_STROKE_SCALE = 0.75

/** Builds a checkerboard circle icon rendered on a canvas. */
export function buildEndStyleCanvas(): HTMLCanvasElement {
  const size = 18
  const canvas = document.createElement("canvas")
  canvas.width = size
  canvas.height = size
  const ctx = canvas.getContext("2d")!
  const cx = size / 2
  const r = size / 2 - 1

  // White base circle.
  ctx.beginPath()
  ctx.arc(cx, cx, r, 0, Math.PI * 2)
  ctx.fillStyle = "#ffffff"
  ctx.fill()

  // Clip to inner circle, draw 4x4 checkerboard.
  ctx.save()
  ctx.beginPath()
  ctx.arc(cx, cx, r - 1, 0, Math.PI * 2)
  ctx.clip()
  const cell = (size - 4) / 4
  for (let row = 0; row < 4; row++) {
    for (let col = 0; col < 4; col++) {
      ctx.fillStyle = (row + col) % 2 === 0 ? "#1f2937" : "#ffffff"
      ctx.fillRect(2 + col * cell, 2 + row * cell, cell, cell)
    }
  }
  ctx.restore()

  // White border ring.
  ctx.beginPath()
  ctx.arc(cx, cx, r, 0, Math.PI * 2)
  ctx.strokeStyle = "#ffffff"
  ctx.lineWidth = 2
  ctx.stroke()

  return canvas
}

/**
 * Traces the chevron path on `ctx` in CSS-pixel coordinates: two arms meeting
 * at an apex near the top of the canvas (= "north" when the icon is unrotated).
 * Symmetric around the vertical centerline so anchoring at [0.5, 0.5] places
 * the chevron's geometric center on the track point.
 */
export function traceArrowPath(ctx: CanvasRenderingContext2D): void {
  const size = ARROW_CANVAS_SIZE
  const cx = size / 2
  const apexY = TRACK_LINE_HALO_WIDTH / 2 + 1
  const armY = size - TRACK_LINE_HALO_WIDTH / 2 - 1
  const armDx = 5
  ctx.beginPath()
  ctx.moveTo(cx - armDx, armY)
  ctx.lineTo(cx, apexY)
  ctx.lineTo(cx + armDx, armY)
}

/**
 * Creates a high-DPR canvas sized `ARROW_CANVAS_SIZE` CSS pixels with the
 * drawing transform pre-scaled by `dpr`, so subsequent path commands use
 * logical (CSS) coordinates and rasterize at full device resolution. Pair with
 * `scale: 1 / dpr` on the OL `Icon` to display crisp pixels on hi-DPI screens.
 */
export function createArrowCanvas(dpr: number): HTMLCanvasElement {
  const canvas = document.createElement("canvas")
  canvas.width = ARROW_CANVAS_SIZE * dpr
  canvas.height = ARROW_CANVAS_SIZE * dpr
  const ctx = canvas.getContext("2d")!
  ctx.scale(dpr, dpr)
  ctx.lineCap = "round"
  ctx.lineJoin = "round"
  return canvas
}

/**
 * Builds the chevron halo (white outer stroke) sprite. Rendered on its own
 * layer below the colored inner stroke so all white halos in the scene (track
 * + every chevron) compose into one continuous silhouette rather than crossing
 * each other.
 */
export function buildArrowHaloCanvas(dpr: number): HTMLCanvasElement {
  const canvas = createArrowCanvas(dpr)
  const ctx = canvas.getContext("2d")!
  ctx.lineWidth = TRACK_LINE_HALO_WIDTH * ARROW_STROKE_SCALE
  ctx.strokeStyle = "#ffffff"
  traceArrowPath(ctx)
  ctx.stroke()
  return canvas
}

/**
 * Builds the chevron inner (colored) sprite. Drawn on top of all halos so it
 * matches the track inner stroke and blends through the chevron apex into the
 * track line.
 */
export function buildArrowInnerCanvas(
  color: string,
  dpr: number
): HTMLCanvasElement {
  const canvas = createArrowCanvas(dpr)
  const ctx = canvas.getContext("2d")!
  ctx.lineWidth = TRACK_LINE_INNER_WIDTH * ARROW_STROKE_SCALE
  ctx.strokeStyle = color
  traceArrowPath(ctx)
  ctx.stroke()
  return canvas
}
