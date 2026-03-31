package api

import (
	"bytes"
	"database/sql"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/uber/h3-go/v4"
	"jo-m.ch/go/cartomancer/internal/pkg/blob"
	"jo-m.ch/go/cartomancer/internal/pkg/load"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/roadclosures"
	"jo-m.ch/go/cartomancer/internal/pkg/session"
	"jo-m.ch/go/cartomancer/internal/pkg/track"
)

type roadClosureResponse struct {
	UUID            string              `json:"uuid"`
	Type            string              `json:"type"`
	StartsAt        *string             `json:"startsAt"`
	EndsAt          *string             `json:"endsAt"`
	Title           string              `json:"title"`
	Reason          *string             `json:"reason"`
	Description     *string             `json:"description"`
	ContentProvider *string             `json:"contentProvider"`
	Geometry        string              `json:"geometry"`
	Attribution     attributionResponse `json:"attribution"`
}

// handleGetTrackRoadClosures returns all active road closures that intersect a track.
// Coarse filtering is done via H3 res-7 cells in the DB, then fine filtering
// at H3 res-12 confirms that each closure actually overlaps the track.
func (sv *server) handleGetTrackRoadClosures(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.GetUser(ctx)
	trackUUID := chi.URLParam(r, "uuid")

	t, err := sv.d.QueryRO().GetTrackByUUID(ctx, trackUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "track not found")
			return
		}
		logg.Error(ctx, "failed to get track", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	if t.Public == 0 && (user == nil || user.Uuid != t.UserID) {
		writeError(w, http.StatusNotFound, "track not found")
		return
	}

	b, err := blob.Get(ctx, sv.d.QueryRO(), t.BlobID)
	if err != nil {
		logg.Error(ctx, "failed to get track blob", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	src, err := load.Blob(t.OriginalFilename, bytes.NewReader(b.Content))
	if err != nil {
		logg.Error(ctx, "failed to parse track blob", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	// Collect all track points and their coarse H3 cells.
	tr, err := track.New(src, 0)
	if err != nil {
		logg.Error(ctx, "failed to create track", "err", err)
		writeStatusError(w, http.StatusUnprocessableEntity)
		return
	}
	pts := tr.Points()
	cellSet := make(map[h3.Cell]struct{})
	lats := make([]float64, len(pts))
	lons := make([]float64, len(pts))
	for i, p := range pts {
		lats[i] = p.Lat
		lons[i] = p.Lon
		cell, cellErr := h3.LatLngToCell(h3.LatLng{Lat: p.Lat, Lng: p.Lon}, roadclosures.CellResolution)
		if cellErr != nil {
			continue
		}
		cellSet[cell] = struct{}{}
	}

	cells := make([]int64, 0, len(cellSet))
	for c := range cellSet {
		cells = append(cells, int64(c))
	}

	// Coarse DB lookup: closures sharing at least one res-7 cell with the track.
	candidates, err := sv.d.GetActiveRoadClosuresByCells(ctx, cells)
	if err != nil {
		logg.Error(ctx, "failed to query road closures by cells", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	// Fine filter: check actual intersection at res-12.
	var result []roadClosureResponse
	for _, rc := range candidates {
		if !roadclosures.Intersects(rc.Geometry, lats, lons) {
			continue
		}
		result = append(result, roadClosureResponse{
			UUID:            rc.Uuid,
			Type:            rc.Type,
			StartsAt:        nullTimeStr(rc.StartsAt),
			EndsAt:          nullTimeStr(rc.EndsAt),
			Title:           rc.Title,
			Reason:          nullStr(rc.Reason),
			Description:     nullStr(rc.Description),
			ContentProvider: nullStr(rc.ContentProvider),
			Geometry:        rc.Geometry,
			Attribution:     attributionResponse{Text: rc.Attribution, Href: rc.AttributionHref},
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"closures": result})
}

// nullStr converts a sql.NullString to a *string.
func nullStr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

// nullTimeStr converts a sql.NullTime to a *string (RFC 3339 date).
func nullTimeStr(nt sql.NullTime) *string {
	if !nt.Valid {
		return nil
	}
	s := nt.Time.Format("2006-01-02")
	return &s
}
