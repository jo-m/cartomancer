/**
 * Stroke widths used for track polylines on OpenLayers maps. Shared between
 * the single-track detail view and the multi-track filter map view so the
 * lines render at a consistent thickness everywhere.
 *
 * A track line is drawn as two stacked strokes: a wider white halo for
 * contrast against the basemap, and a narrower inner stroke in the track's
 * color.
 */

/** Width in pixels of the white halo behind each track line. */
export const TRACK_LINE_HALO_WIDTH = 7

/** Width in pixels of the colored inner stroke of each track line. */
export const TRACK_LINE_INNER_WIDTH = 4
