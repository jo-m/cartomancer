package api

import (
	"database/sql"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/maps"
)

// mapBuildResponse is the JSON representation of a ready map build.
type mapBuildResponse struct {
	UUID       string    `json:"uuid"`
	Key        string    `json:"key"`
	Version    string    `json:"version"`
	Uploaded   time.Time `json:"uploaded"`
	Size       int64     `json:"size"`
	LocalSize  *int64    `json:"localSize,omitempty"`
	MaxZoom    int64     `json:"maxZoom"`
	BboxMinLon *float64  `json:"bboxMinLon,omitempty"`
	BboxMinLat *float64  `json:"bboxMinLat,omitempty"`
	BboxMaxLon *float64  `json:"bboxMaxLon,omitempty"`
	BboxMaxLat *float64  `json:"bboxMaxLat,omitempty"`
}

// adminMapBuildResponse is the JSON representation of a map build for admin endpoints,
// including status fields not exposed on the public API.
type adminMapBuildResponse struct {
	mapBuildResponse
	CreatedAt         time.Time `json:"createdAt"`
	Ready             bool      `json:"ready"`
	MarkedForDeletion bool      `json:"markedForDeletion"`
}

// toMapBuildResponse converts a db.MapBuild to the public JSON response type.
func toMapBuildResponse(b db.MapBuild) mapBuildResponse {
	r := mapBuildResponse{
		UUID:     b.Uuid,
		Key:      b.Key,
		Version:  b.Version,
		Uploaded: b.Uploaded,
		Size:     b.Size,
		MaxZoom:  b.Maxzoom,
	}
	if b.LocalSize.Valid {
		v := b.LocalSize.Int64
		r.LocalSize = &v
	}
	if b.BboxMinLon.Valid {
		v := b.BboxMinLon.Float64
		r.BboxMinLon = &v
	}
	if b.BboxMinLat.Valid {
		v := b.BboxMinLat.Float64
		r.BboxMinLat = &v
	}
	if b.BboxMaxLon.Valid {
		v := b.BboxMaxLon.Float64
		r.BboxMaxLon = &v
	}
	if b.BboxMaxLat.Valid {
		v := b.BboxMaxLat.Float64
		r.BboxMaxLat = &v
	}
	return r
}

// toAdminMapBuildResponse converts a db.MapBuild to the admin JSON response type.
func toAdminMapBuildResponse(b db.MapBuild) adminMapBuildResponse {
	return adminMapBuildResponse{
		mapBuildResponse:  toMapBuildResponse(b),
		CreatedAt:         b.CreatedAt,
		Ready:             b.Ready != 0,
		MarkedForDeletion: b.MarkedForDeletion != 0,
	}
}

// handleListMapBuilds returns all ready PMTiles map builds.
func (sv *server) handleListMapBuilds(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	builds, err := sv.d.QueryRO().ListReadyMapBuilds(ctx)
	if err != nil {
		logg.Error(ctx, "failed to list map builds", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	resp := make([]mapBuildResponse, len(builds))
	for i, b := range builds {
		resp[i] = toMapBuildResponse(b)
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleAdminListMapBuilds returns all map builds regardless of status.
func (sv *server) handleAdminListMapBuilds(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	builds, err := sv.d.QueryRO().ListMapBuilds(ctx)
	if err != nil {
		logg.Error(ctx, "failed to list map builds", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	resp := make([]adminMapBuildResponse, len(builds))
	for i, b := range builds {
		resp[i] = toAdminMapBuildResponse(b)
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleAdminMarkMapForDeletion sets the marked_for_deletion flag on a map build.
func (sv *server) handleAdminMarkMapForDeletion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uuid := chi.URLParam(r, "uuid")

	_, err := sv.d.QueryRW().SetMapBuildMarkedForDeletion(ctx, uuid)
	if err != nil {
		logg.Error(ctx, "failed to mark map build for deletion", "err", err, "uuid", uuid)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGetMapFile serves a PMTiles file by UUID, supporting HTTP range requests.
func (sv *server) handleGetMapFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uuid := chi.URLParam(r, "uuid")

	build, err := sv.d.QueryRO().GetReadyMapBuildByUUID(ctx, uuid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "map not found")
			return
		}
		logg.Error(ctx, "failed to get map build", "err", err, "uuid", uuid)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	path := maps.OutputPath(sv.mapsDir, build.Uuid)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "map file not found")
			return
		}
		logg.Error(ctx, "failed to open map file", "err", err, "path", path)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		logg.Error(ctx, "failed to stat map file", "err", err, "path", path)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.pmtiles")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeContent(w, r, build.Uuid+".pmtiles", stat.ModTime(), f)
}
