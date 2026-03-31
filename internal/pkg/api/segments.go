package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
)

type segmentEntry struct {
	UUID         string  `json:"uuid"`
	DistanceM    float64 `json:"distanceM"`
	AscentM      float64 `json:"ascentM"`
	NTracks      int64   `json:"nTracks"`
	H3Resolution int64   `json:"h3Resolution"`
	Polyline     string  `json:"polyline"`
	StartLat     float64 `json:"startLat"`
	StartLon     float64 `json:"startLon"`
	EndLat       float64 `json:"endLat"`
	EndLon       float64 `json:"endLon"`
}

type listSegmentsResponse struct {
	Segments []segmentEntry `json:"segments"`
}

type segmentJunction struct {
	UUID   string  `json:"uuid"`
	H3Cell string  `json:"h3Cell"`
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
}

type segmentDetailResponse struct {
	UUID          string          `json:"uuid"`
	DistanceM     float64         `json:"distanceM"`
	AscentM       float64         `json:"ascentM"`
	NTracks       int64           `json:"nTracks"`
	H3Resolution  int64           `json:"h3Resolution"`
	Polyline      string          `json:"polyline"`
	StartJunction segmentJunction `json:"startJunction"`
	EndJunction   segmentJunction `json:"endJunction"`
	TrackUUIDs    []string        `json:"trackUuids"`
}

type listJunctionsResponse struct {
	Junctions []segmentJunction `json:"junctions"`
}

// handleListSegments returns all segments with their polylines for map display.
func (sv *server) handleListSegments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rows, err := sv.d.QueryRO().ListAllSegments(ctx)
	if err != nil {
		logg.Error(ctx, "failed to list segments", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	entries := make([]segmentEntry, len(rows))
	for i, row := range rows {
		entries[i] = segmentEntry{
			UUID:         row.Uuid,
			DistanceM:    row.DistanceM,
			AscentM:      row.AscentM,
			NTracks:      row.NTracks,
			H3Resolution: row.H3Resolution,
			Polyline:     row.Polyline,
			StartLat:     row.StartLat,
			StartLon:     row.StartLon,
			EndLat:       row.EndLat,
			EndLon:       row.EndLon,
		}
	}

	writeJSON(w, http.StatusOK, listSegmentsResponse{Segments: entries})
}

// handleGetSegment returns the detail of a single segment including its
// junctions, polyline, and member track UUIDs.
func (sv *server) handleGetSegment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	segUUID := chi.URLParam(r, "uuid")

	seg, err := sv.d.QueryRO().GetSegment(ctx, segUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "segment not found")
			return
		}
		logg.Error(ctx, "failed to get segment", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	trackUUIDs, err := sv.d.QueryRO().ListSegmentTrackUUIDs(ctx, segUUID)
	if err != nil {
		logg.Error(ctx, "failed to list segment tracks", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}
	if trackUUIDs == nil {
		trackUUIDs = []string{}
	}

	writeJSON(w, http.StatusOK, segmentDetailResponse{
		UUID:         seg.Uuid,
		DistanceM:    seg.DistanceM,
		AscentM:      seg.AscentM,
		NTracks:      seg.NTracks,
		H3Resolution: seg.H3Resolution,
		Polyline:     seg.Polyline,
		StartJunction: segmentJunction{
			UUID:   seg.StartJunctionID,
			H3Cell: seg.StartH3Cell,
			Lat:    seg.StartLat,
			Lon:    seg.StartLon,
		},
		EndJunction: segmentJunction{
			UUID:   seg.EndJunctionID,
			H3Cell: seg.EndH3Cell,
			Lat:    seg.EndLat,
			Lon:    seg.EndLon,
		},
		TrackUUIDs: trackUUIDs,
	})
}

// handleListSegmentJunctions returns all segment junctions for map display.
func (sv *server) handleListSegmentJunctions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rows, err := sv.d.QueryRO().ListAllSegmentJunctions(ctx)
	if err != nil {
		logg.Error(ctx, "failed to list junctions", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	junctions := make([]segmentJunction, len(rows))
	for i, row := range rows {
		junctions[i] = segmentJunction{
			UUID:   row.Uuid,
			H3Cell: row.H3Cell,
			Lat:    row.Lat,
			Lon:    row.Lon,
		}
	}

	writeJSON(w, http.StatusOK, listJunctionsResponse{Junctions: junctions})
}
