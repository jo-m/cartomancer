package api

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"jo-m.ch/go/cartomancer/internal/pkg/blob"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/forecast"
	"jo-m.ch/go/cartomancer/internal/pkg/geonames"
	"jo-m.ch/go/cartomancer/internal/pkg/jobs"
	"jo-m.ch/go/cartomancer/internal/pkg/load"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/session"
	"jo-m.ch/go/cartomancer/internal/pkg/track"
	"jo-m.ch/go/cartomancer/internal/pkg/trackgroup"
)

type trackResponse struct {
	UUID                    string                 `json:"uuid"`
	Name                    string                 `json:"name"`
	Description             string                 `json:"description,omitempty"`
	Source                  string                 `json:"source,omitempty"`
	Author                  string                 `json:"author,omitempty"`
	AuthorLinkURL           string                 `json:"authorLinkUrl,omitempty"`
	FileFormat              int                    `json:"fileFormat"`
	OriginalFilename        string                 `json:"originalFilename,omitempty"`
	TrackType               int                    `json:"trackType"`
	LinkURL                 string                 `json:"linkUrl,omitempty"`
	Sport                   int                    `json:"sport"`
	SubSport                int                    `json:"subSport"`
	TotalDistanceM          float64                `json:"totalDistanceM"`
	TotalAscentM            float64                `json:"totalAscentM"`
	Elevation               *minMaxResponse        `json:"elevation,omitempty"`
	Start                   *lonLatResponse        `json:"start,omitempty"`
	End                     *lonLatResponse        `json:"end,omitempty"`
	Bounds                  *bboxResponse          `json:"bounds,omitempty"`
	OriginalCreatedAt       string                 `json:"originalCreatedAt,omitempty"`
	CreatedAt               string                 `json:"createdAt"`
	UpdatedAt               string                 `json:"updatedAt"`
	Public                  bool                   `json:"public"`
	InitialEditingCompleted bool                   `json:"initialEditingCompleted"`
	Starred                 bool                   `json:"starred"`
	IsOwner                 bool                   `json:"isOwner"`
	User                    userRefResponse        `json:"user"`
	Tags                    []string               `json:"tags"`
	GeonameLabel            string                 `json:"geonameLabel,omitempty"`
	SimilarTracks           []similarTrackEntry    `json:"similarTracks"`
	Forecast                *trackForecastResponse `json:"forecast,omitempty"`
}

type trackForecastResponse struct {
	ForecastReferenceTime string   `json:"forecastReferenceTime"`
	StartTime             string   `json:"startTime"`
	AvgTemperatureC       *float64 `json:"avgTemperatureC,omitempty"`
	TotalPrecipitationMm  *float64 `json:"totalPrecipitationMm,omitempty"`
	WindHeadMs            *float64 `json:"windHeadMs,omitempty"`
	WindRightMs           *float64 `json:"windRightMs,omitempty"`
	WindTailMs            *float64 `json:"windTailMs,omitempty"`
	WindLeftMs            *float64 `json:"windLeftMs,omitempty"`
}

type similarTrackEntry struct {
	UUID           string  `json:"uuid"`
	Name           string  `json:"name"`
	TotalDistanceM float64 `json:"totalDistanceM"`
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{Valid: true, String: s}
}

func toNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Valid: true, Time: *t}
}

func toNullFloat64(f *float64) sql.NullFloat64 {
	if f == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Valid: true, Float64: *f}
}

func nullFloat64Ptr(f sql.NullFloat64) *float64 {
	if f.Valid {
		return &f.Float64
	}
	return nil
}

func nullStringVal(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// splitTags parses a JSON array of tag strings (from json_group_array) into a slice.
// Returns an empty (non-nil) slice for empty input or "[]".
func splitTags(jsonArr string) []string {
	if jsonArr == "" || jsonArr == "[]" {
		return []string{}
	}
	var tags []string
	if err := json.Unmarshal([]byte(jsonArr), &tags); err != nil {
		return []string{}
	}
	return tags
}

func trackResponseFromDB(tw db.TrackWithStarred, tags []string, similar []db.GetSimilarTracksRow, isOwner bool) trackResponse {
	if tags == nil {
		tags = []string{}
	}
	simEntries := make([]similarTrackEntry, len(similar))
	for i, s := range similar {
		simEntries[i] = similarTrackEntry{
			UUID:           s.Uuid,
			Name:           s.Name,
			TotalDistanceM: s.TotalDistanceM,
		}
	}
	t := tw.Track
	resp := trackResponse{
		UUID:                    t.Uuid,
		Name:                    t.Name,
		Description:             nullStringVal(t.Description),
		Source:                  nullStringVal(t.Source),
		Author:                  nullStringVal(t.Author),
		AuthorLinkURL:           nullStringVal(t.AuthorLinkUrl),
		FileFormat:              int(t.FileFormat),
		OriginalFilename:        t.OriginalFilename,
		TrackType:               int(t.TrackType),
		LinkURL:                 nullStringVal(t.LinkUrl),
		Sport:                   int(t.Sport),
		SubSport:                int(t.SubSport),
		TotalDistanceM:          t.TotalDistanceM,
		TotalAscentM:            t.TotalAscentM,
		Elevation:               nullMinMax(t.MinElevationM, t.MaxElevationM),
		Start:                   nullLonLat(t.StartLon, t.StartLat),
		End:                     nullLonLat(t.EndLon, t.EndLat),
		Bounds:                  nullBBox(t.BoundsMinLat, t.BoundsMinLon, t.BoundsMaxLat, t.BoundsMaxLon),
		CreatedAt:               t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:               t.UpdatedAt.Format(time.RFC3339),
		Public:                  t.Public != 0,
		InitialEditingCompleted: t.InitialEditingCompleted != 0,
		Starred:                 tw.Starred,
		IsOwner:                 isOwner,
		User:                    userRefResponse{UUID: t.UserID, Name: tw.UserName},
		Tags:                    tags,
		GeonameLabel:            tw.GeonameLabel,
		SimilarTracks:           simEntries,
	}
	if t.OriginalCreatedAt.Valid {
		resp.OriginalCreatedAt = t.OriginalCreatedAt.Time.Format(time.RFC3339)
	}
	if tw.Forecast.HasData() {
		resp.Forecast = &trackForecastResponse{
			ForecastReferenceTime: tw.Forecast.ForecastReferenceTime.Time.Format(time.RFC3339),
			StartTime:             tw.Forecast.StartTime.Time.Format(time.RFC3339),
			AvgTemperatureC:       nullFloat64Ptr(tw.Forecast.AvgTemperatureC),
			TotalPrecipitationMm:  nullFloat64Ptr(tw.Forecast.TotalPrecipitationMm),
			WindHeadMs:            nullFloat64Ptr(tw.Forecast.WindHeadMs),
			WindRightMs:           nullFloat64Ptr(tw.Forecast.WindRightMs),
			WindTailMs:            nullFloat64Ptr(tw.Forecast.WindTailMs),
			WindLeftMs:            nullFloat64Ptr(tw.Forecast.WindLeftMs),
		}
	}
	return resp
}

func fileFormatFromExt(filename string) track.FileFormat {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".fit":
		return track.FileFormatFIT
	case ".gpx":
		return track.FileFormatGPX
	default:
		return track.FileFormatGPX
	}
}

// trackVisibleToUser reports whether t is accessible to user.
// A track is accessible if it is public or user is the owner.
// user may be nil for anonymous requests.
func trackVisibleToUser(t db.Track, user *db.User) bool {
	return t.Public != 0 || (user != nil && user.Uuid == t.UserID)
}

// getViewableTrack fetches a track by UUID and checks that the requesting user
// may view it. On failure it writes the appropriate error response and returns
// (db.Track{}, false).
func (sv *server) getViewableTrack(w http.ResponseWriter, r *http.Request, trackUUID string) (db.Track, bool) {
	ctx := r.Context()
	user := session.GetUser(ctx)

	t, err := sv.d.QueryRO().GetTrackByUUID(ctx, trackUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "track not found")
			return db.Track{}, false
		}
		logg.Error(ctx, "failed to get track", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return db.Track{}, false
	}

	if !trackVisibleToUser(t, user) {
		writeError(w, http.StatusNotFound, "track not found")
		return db.Track{}, false
	}

	return t, true
}

func (sv *server) handleGetTrack(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.GetUser(ctx)
	trackUUID := chi.URLParam(r, "uuid")

	var viewerID string
	if user != nil {
		viewerID = user.Uuid
	}

	t, err := sv.d.GetTrackByUUIDForViewer(ctx, trackUUID, viewerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "track not found")
			return
		}
		logg.Error(ctx, "failed to get track", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	if !trackVisibleToUser(t.Track, user) {
		writeError(w, http.StatusNotFound, "track not found")
		return
	}

	tags, err := sv.d.QueryRO().GetTagsByTrackID(ctx, trackUUID)
	if err != nil {
		logg.Error(ctx, "failed to get track tags", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	similar, err := sv.d.QueryRO().GetSimilarTracks(ctx, trackUUID)
	if err != nil {
		logg.Error(ctx, "failed to get similar tracks", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, trackResponseFromDB(t, tags, similar, user != nil && user.Uuid == t.UserID))
}

func (sv *server) handleDownloadTrackBlob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	trackUUID := chi.URLParam(r, "uuid")

	t, ok := sv.getViewableTrack(w, r, trackUUID)
	if !ok {
		return
	}

	var contentType, ext string
	switch track.FileFormat(t.FileFormat) {
	case track.FileFormatGPX:
		contentType = "application/gpx+xml"
		ext = ".gpx"
	case track.FileFormatFIT:
		contentType = "application/vnd.ant.fit"
		ext = ".fit"
	default:
		contentType = "application/octet-stream"
		ext = ".bin"
	}

	if err := blob.Serve(w, r, sv.d.QueryRO(), t.BlobID, contentType, t.Name+ext); err != nil {
		logg.Error(ctx, "failed to serve blob", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}
}

func (sv *server) handleDownloadTrackSVG(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	trackUUID := chi.URLParam(r, "uuid")

	t, ok := sv.getViewableTrack(w, r, trackUUID)
	if !ok {
		return
	}

	opts, err := parseSVGOptions(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	eTag := fmt.Sprintf(`"%d-%d-v2"`, t.UpdatedAt.UnixMilli(), opts.Size)
	if r.Header.Get(headerIfNoneMatch) == eTag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	pts, err := loadViewerPoints(t, db.PreviewPolyline50M)
	if err != nil {
		logg.Error(ctx, "failed to load preview points", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	var bounds *track.Bounds
	if t.BoundsMinLat.Valid && t.BoundsMinLon.Valid && t.BoundsMaxLat.Valid && t.BoundsMaxLon.Valid {
		bounds = &track.Bounds{
			MinLat: t.BoundsMinLat.Float64,
			MinLon: t.BoundsMinLon.Float64,
			MaxLat: t.BoundsMaxLat.Float64,
			MaxLon: t.BoundsMaxLon.Float64,
		}
	}

	svg := []byte(pts.PreviewSVG(opts, bounds))
	w.Header().Set(headerContentType, "image/svg+xml")
	w.Header().Set(headerCacheControl, "private, max-age=3600")
	w.Header().Set(headerETag, eTag)
	w.Header().Set(headerContentLength, strconv.Itoa(len(svg)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(svg)
}

func (sv *server) handleDownloadTrackProfileSVG(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	trackUUID := chi.URLParam(r, "uuid")

	t, ok := sv.getViewableTrack(w, r, trackUUID)
	if !ok {
		return
	}

	opts, err := parseSVGOptions(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	eTag := fmt.Sprintf(`"%d-%d-v3"`, t.UpdatedAt.UnixMilli(), opts.Size)
	if r.Header.Get(headerIfNoneMatch) == eTag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	pts, err := loadViewerPoints(t, db.PreviewPolyline5M)
	if err != nil {
		logg.Error(ctx, "failed to load profile points", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	svg := []byte(pts.ProfileSVG(opts))
	w.Header().Set(headerContentType, "image/svg+xml")
	w.Header().Set(headerCacheControl, "private, max-age=3600")
	w.Header().Set(headerETag, eTag)
	w.Header().Set(headerContentLength, strconv.Itoa(len(svg)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(svg)
}

// parseSVGOptions reads the optional size query parameter from r and returns a
// PreviewOptions struct, falling back to defaults for missing params.
// The color is always "currentColor" so the frontend can override it via CSS.
func parseSVGOptions(r *http.Request) (track.PreviewOptions, error) {
	opts := track.DefaultPreviewOptions()
	q := r.URL.Query()

	if s := q.Get("size"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v < 16 || v > 512 {
			return opts, fmt.Errorf("size must be an integer between 16 and 512")
		}
		opts.Size = v
	}

	return opts, nil
}

type trackPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
	Ele float64 `json:"ele"`
	D   float64 `json:"d"`
}

// handleGetTrackPoints returns track points with elevation and cumulative distance.
// Public tracks are accessible without authentication; private tracks require the owner.
func (sv *server) handleGetTrackPoints(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	trackUUID := chi.URLParam(r, "uuid")

	t, ok := sv.getViewableTrack(w, r, trackUUID)
	if !ok {
		return
	}

	eTag := fmt.Sprintf(`"%d-points-v5"`, t.UpdatedAt.UnixMilli())
	if r.Header.Get(headerIfNoneMatch) == eTag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	pts, err := loadViewerPoints(t, db.PreviewPolyline5M)
	if err != nil {
		logg.Error(ctx, "failed to load viewer points", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	points := make([]trackPoint, len(pts))
	for i, p := range pts {
		points[i] = trackPoint{Lat: p.Lat, Lon: p.Lon, Ele: p.Elevation, D: p.Distance}
	}

	w.Header().Set(headerCacheControl, "private, no-cache")
	w.Header().Set(headerETag, eTag)
	writeJSON(w, http.StatusOK, map[string]any{"points": points})
}

type editTrackRequest struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Source        string   `json:"source"`
	Author        string   `json:"author"`
	AuthorLinkURL string   `json:"authorLinkUrl"`
	TrackType     int64    `json:"trackType"`
	LinkURL       string   `json:"linkUrl"`
	Sport         int64    `json:"sport"`
	SubSport      int64    `json:"subSport"`
	Public        bool     `json:"public"`
	Tags          []string `json:"tags"`
}

func (sv *server) handleEditTrack(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.MustGetUser(ctx)
	trackUUID := chi.URLParam(r, "uuid")

	var req editTrackRequest
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	for _, tag := range req.Tags {
		if !validateTag(tag) {
			writeError(w, http.StatusBadRequest, "each tag must be 2-32 alphanumeric characters")
			return
		}
	}

	now := time.Now().UTC()
	var updated db.Track
	err := sv.d.WithTx(ctx, func(q *db.Queries) error {
		existing, txErr := q.GetTrackByUUID(ctx, trackUUID)
		if txErr != nil {
			return txErr
		}
		if existing.UserID != user.Uuid {
			return errForbidden
		}

		var public int64
		if req.Public {
			public = 1
		}
		updated, txErr = q.UpdateTrack(ctx, db.UpdateTrackParams{
			UpdatedAt:     now,
			Name:          req.Name,
			Description:   toNullString(req.Description),
			Source:        toNullString(req.Source),
			Author:        toNullString(req.Author),
			AuthorLinkUrl: toNullString(req.AuthorLinkURL),
			TrackType:     req.TrackType,
			LinkUrl:       toNullString(req.LinkURL),
			Sport:         req.Sport,
			SubSport:      req.SubSport,
			Public:        public,
			Uuid:          trackUUID,
		})
		if txErr != nil {
			return txErr
		}

		if txErr = q.DeleteTrackTags(ctx, trackUUID); txErr != nil {
			return txErr
		}
		for _, tag := range req.Tags {
			t, txErr := q.UpsertTag(ctx, db.UpsertTagParams{Tag: tag, UserID: user.Uuid})
			if txErr != nil {
				return txErr
			}
			if txErr = q.CreateTrackTag(ctx, db.CreateTrackTagParams{
				TrackID: trackUUID,
				TagID:   t.ID,
			}); txErr != nil {
				return txErr
			}
		}
		return nil
	})
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "track not found")
		return
	}
	if errors.Is(err, errForbidden) {
		writeStatusError(w, http.StatusForbidden)
		return
	}
	if err != nil {
		logg.Error(ctx, "failed to update track", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	starCount, err := sv.d.QueryRO().IsTrackStarredByUser(ctx, db.IsTrackStarredByUserParams{
		TrackID: trackUUID,
		UserID:  user.Uuid,
	})
	if err != nil {
		logg.Error(ctx, "failed to get starred status", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, trackResponseFromDB(db.TrackWithStarred{
		Track:          updated,
		Starred:        starCount > 0,
		UserName:       user.Name,
		UserAvatarSeed: user.AvatarSeed,
	}, req.Tags, nil, true))
}

// handleDeleteTrack handles DELETE /tracks/{uuid}.
// Deletes the track and all associated data (tags, stars, blob).
// Only the track owner may delete it.
func (sv *server) handleDeleteTrack(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.MustGetUser(ctx)
	trackUUID := chi.URLParam(r, "uuid")

	err := sv.d.WithTx(ctx, func(q *db.Queries) error {
		existing, txErr := q.GetTrackByUUID(ctx, trackUUID)
		if txErr != nil {
			return txErr
		}
		if existing.UserID != user.Uuid {
			return errForbidden
		}
		return q.DeleteTrack(ctx, trackUUID)
	})
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "track not found")
		return
	}
	if errors.Is(err, errForbidden) {
		writeStatusError(w, http.StatusForbidden)
		return
	}
	if err != nil {
		logg.Error(ctx, "failed to delete track", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type bulkDeleteTracksRequest struct {
	UUIDs []string `json:"uuids"`
}

// handleBulkDeleteTracks handles POST /tracks/bulk-delete.
// Deletes multiple tracks and all associated data in a single operation.
// All tracks must belong to the authenticated user.
func (sv *server) handleBulkDeleteTracks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.MustGetUser(ctx)

	var req bulkDeleteTracksRequest
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	if len(req.UUIDs) == 0 {
		writeError(w, http.StatusBadRequest, "uuids is required and must not be empty")
		return
	}
	if len(req.UUIDs) > maxBulkEditUUIDs {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("at most %d uuids allowed", maxBulkEditUUIDs))
		return
	}

	err := sv.d.BulkDeleteTracks(ctx, req.UUIDs, user.Uuid)
	if errors.Is(err, db.ErrBulkDeleteMismatch) {
		writeError(w, http.StatusNotFound, "one or more tracks not found or not owned by you")
		return
	}
	if err != nil {
		logg.Error(ctx, "failed to bulk delete tracks", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type listTracksResponse struct {
	Tracks     []trackResponse `json:"tracks"`
	TotalCount int             `json:"totalCount"`
	Page       int             `json:"page"`
	PageSize   int             `json:"pageSize"`
}

func parseOptionalFloat64(q map[string][]string, key string) (*float64, error) {
	vals, ok := q[key]
	if !ok || len(vals) == 0 || vals[0] == "" {
		return nil, nil
	}
	v, err := strconv.ParseFloat(vals[0], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid value for %q: %w", key, err)
	}
	return &v, nil
}

func parseOptionalTime(q map[string][]string, key string) (*time.Time, error) {
	vals, ok := q[key]
	if !ok || len(vals) == 0 || vals[0] == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, vals[0])
	if err != nil {
		return nil, fmt.Errorf("invalid value for %q: %w", key, err)
	}
	return &t, nil
}

func parseOptionalString(q map[string][]string, key string) *string {
	vals, ok := q[key]
	if !ok || len(vals) == 0 || vals[0] == "" {
		return nil
	}
	return &vals[0]
}

func parseInt64Slice(q map[string][]string, key string) ([]int64, error) {
	vals := q[key]
	if len(vals) == 0 {
		return nil, nil
	}
	result := make([]int64, 0, len(vals))
	for _, v := range vals {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value for %q: %w", key, err)
		}
		result = append(result, n)
	}
	return result, nil
}

// parseListTracksParams parses the shared track-listing query string used by
// both [server.handleListTracks] and [server.handleListTrackPolylines]. On
// validation failure it returns a non-nil error whose message is safe to
// surface as an HTTP 400 body.
func parseListTracksParams(r *http.Request) (db.ListTracksParams, error) {
	ctx := r.Context()
	user := session.GetUser(ctx)

	q := r.URL.Query()
	qmap := map[string][]string(q)

	var params db.ListTracksParams
	if user != nil {
		params.UserID = user.Uuid
		params.ViewerUserID = user.Uuid
	}

	if v := q.Get("onlyMine"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return params, fmt.Errorf("invalid value for 'onlyMine'")
		}
		if b && user != nil {
			params.OnlyOwnedByUser = true
		}
	}

	if v := q.Get("onlyStarred"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return params, fmt.Errorf("invalid value for 'onlyStarred'")
		}
		if b && user != nil {
			params.OnlyStarred = true
		}
	}

	if v := q.Get("page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return params, fmt.Errorf("invalid value for 'page'")
		}
		params.Page = n
	}
	if v := q.Get("pageSize"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 200 {
			return params, fmt.Errorf("invalid value for 'pageSize': must be 1-200")
		}
		params.PageSize = n
	}

	if v := q.Get("public"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return params, fmt.Errorf("invalid value for 'public'")
		}
		params.Public = &b
	}

	var err error
	if params.FileFormats, err = parseInt64Slice(qmap, "fileFormat"); err != nil {
		return params, err
	}
	if params.TrackTypes, err = parseInt64Slice(qmap, "trackType"); err != nil {
		return params, err
	}
	if params.Sports, err = parseInt64Slice(qmap, "sport"); err != nil {
		return params, err
	}
	if params.SubSports, err = parseInt64Slice(qmap, "subSport"); err != nil {
		return params, err
	}

	if tags := qmap["tag"]; len(tags) > 0 {
		params.Tags = tags
	}
	if v := q.Get("tagsAnd"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return params, fmt.Errorf("invalid value for 'tagsAnd'")
		}
		params.TagsAnd = b
	}

	params.Name = parseOptionalString(qmap, "name")
	params.Description = parseOptionalString(qmap, "description")
	params.Source = parseOptionalString(qmap, "source")

	if params.CreatedAtMin, err = parseOptionalTime(qmap, "createdAtMin"); err != nil {
		return params, err
	}
	if params.CreatedAtMax, err = parseOptionalTime(qmap, "createdAtMax"); err != nil {
		return params, err
	}
	if params.UpdatedAtMin, err = parseOptionalTime(qmap, "updatedAtMin"); err != nil {
		return params, err
	}
	if params.UpdatedAtMax, err = parseOptionalTime(qmap, "updatedAtMax"); err != nil {
		return params, err
	}
	if params.OriginalCreatedAtMin, err = parseOptionalTime(qmap, "originalCreatedAtMin"); err != nil {
		return params, err
	}
	if params.OriginalCreatedAtMax, err = parseOptionalTime(qmap, "originalCreatedAtMax"); err != nil {
		return params, err
	}

	if params.TotalDistanceMMin, err = parseOptionalFloat64(qmap, "totalDistanceMMin"); err != nil {
		return params, err
	}
	if params.TotalDistanceMMax, err = parseOptionalFloat64(qmap, "totalDistanceMMax"); err != nil {
		return params, err
	}
	if params.TotalAscentMMin, err = parseOptionalFloat64(qmap, "totalAscentMMin"); err != nil {
		return params, err
	}
	if params.TotalAscentMMax, err = parseOptionalFloat64(qmap, "totalAscentMMax"); err != nil {
		return params, err
	}

	if params.StartLatMin, err = parseOptionalFloat64(qmap, "startLatMin"); err != nil {
		return params, err
	}
	if params.StartLatMax, err = parseOptionalFloat64(qmap, "startLatMax"); err != nil {
		return params, err
	}
	if params.StartLonMin, err = parseOptionalFloat64(qmap, "startLonMin"); err != nil {
		return params, err
	}
	if params.StartLonMax, err = parseOptionalFloat64(qmap, "startLonMax"); err != nil {
		return params, err
	}
	if params.EndLatMin, err = parseOptionalFloat64(qmap, "endLatMin"); err != nil {
		return params, err
	}
	if params.EndLatMax, err = parseOptionalFloat64(qmap, "endLatMax"); err != nil {
		return params, err
	}
	if params.EndLonMin, err = parseOptionalFloat64(qmap, "endLonMin"); err != nil {
		return params, err
	}
	if params.EndLonMax, err = parseOptionalFloat64(qmap, "endLonMax"); err != nil {
		return params, err
	}

	if params.StartNearLat, err = parseOptionalFloat64(qmap, "startNearLat"); err != nil {
		return params, err
	}
	if params.StartNearLon, err = parseOptionalFloat64(qmap, "startNearLon"); err != nil {
		return params, err
	}
	if params.StartNearRadiusM, err = parseOptionalFloat64(qmap, "startNearRadiusM"); err != nil {
		return params, err
	}
	if params.EndNearLat, err = parseOptionalFloat64(qmap, "endNearLat"); err != nil {
		return params, err
	}
	if params.EndNearLon, err = parseOptionalFloat64(qmap, "endNearLon"); err != nil {
		return params, err
	}
	if params.EndNearRadiusM, err = parseOptionalFloat64(qmap, "endNearRadiusM"); err != nil {
		return params, err
	}

	if v := q.Get("sortBy"); v != "" {
		switch v {
		case "created_at", "total_distance_m", "total_ascent_m":
			params.SortBy = v
		default:
			return params, fmt.Errorf("invalid value for 'sortBy': must be one of created_at, total_distance_m, total_ascent_m")
		}
	}
	if v := q.Get("sortOrder"); v != "" {
		switch v {
		case "asc", "desc":
			params.SortOrder = v
		default:
			return params, fmt.Errorf("invalid value for 'sortOrder': must be asc or desc")
		}
	}

	startNearCount := boolToInt(params.StartNearLat != nil) + boolToInt(params.StartNearLon != nil) + boolToInt(params.StartNearRadiusM != nil)
	if startNearCount > 0 && startNearCount < 3 {
		return params, fmt.Errorf("startNearLat, startNearLon, and startNearRadiusM must all be provided together")
	}
	endNearCount := boolToInt(params.EndNearLat != nil) + boolToInt(params.EndNearLon != nil) + boolToInt(params.EndNearRadiusM != nil)
	if endNearCount > 0 && endNearCount < 3 {
		return params, fmt.Errorf("endNearLat, endNearLon, and endNearRadiusM must all be provided together")
	}

	return params, nil
}

func (sv *server) handleListTracks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.GetUser(ctx)

	params, err := parseListTracksParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := sv.d.ListTracks(ctx, params)
	if err != nil {
		logg.Error(ctx, "failed to list tracks", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	responses := make([]trackResponse, len(result.Tracks))
	for i, t := range result.Tracks {
		tags := splitTags(t.Tags)
		responses[i] = trackResponseFromDB(t, tags, nil, user != nil && user.Uuid == t.UserID)
	}

	if params.Page == 0 {
		params.Page = 1
	}
	if params.PageSize == 0 {
		params.PageSize = 25
	}

	writeJSON(w, http.StatusOK, listTracksResponse{
		Tracks:     responses,
		TotalCount: result.TotalCount,
		Page:       params.Page,
		PageSize:   params.PageSize,
	})
}

// trackPolylineEntry is one row in the response from
// [server.handleListTrackPolylines5M] / [server.handleListTrackPolylines50M].
// The shape is intentionally lean to keep the bulk-listing payload small.
type trackPolylineEntry struct {
	UUID           string        `json:"uuid"`
	Name           string        `json:"name"`
	UserID         string        `json:"userId"`
	UserName       string        `json:"userName"`
	TotalDistanceM float64       `json:"totalDistanceM"`
	TotalAscentM   float64       `json:"totalAscentM"`
	Bounds         *bboxResponse `json:"bounds,omitempty"`
	// Polyline is the simplified track as an array of [lat, lon] pairs in
	// WGS84 degrees, decoded from the stored varint-encoded blob.
	Polyline [][2]float64           `json:"polyline"`
	Starred  bool                   `json:"starred"`
	Forecast *trackForecastResponse `json:"forecast,omitempty"`
}

type listTrackPolylinesResponse struct {
	Tracks     []trackPolylineEntry `json:"tracks"`
	TotalCount int                  `json:"totalCount"`
	Limit      int                  `json:"limit"`
}

// trackPolylinesLimit caps how many polylines a single response carries.
// Tracks in excess of this are reported via TotalCount so the client can
// surface a banner.
const trackPolylinesLimit = 250

// handleListTrackPolylines5M returns 5 m simplified preview polylines.
func (sv *server) handleListTrackPolylines5M(w http.ResponseWriter, r *http.Request) {
	sv.handleListTrackPolylines(w, r, db.PreviewPolyline5M, "5m")
}

// handleListTrackPolylines50M returns 50 m simplified preview polylines.
func (sv *server) handleListTrackPolylines50M(w http.ResponseWriter, r *http.Request) {
	sv.handleListTrackPolylines(w, r, db.PreviewPolyline50M, "50m")
}

// handleListTrackPolylines returns the simplified preview polylines for all
// tracks matching the same query string accepted by [server.handleListTracks].
// Pagination on the input is ignored; instead the server returns up to
// [trackPolylinesLimit] tracks. Strong ETag based on the most recent
// updated_at and the result counts allows the browser to skip the body on
// repeated requests. The varint blob stored in the DB is decoded server-side
// so the response is plain JSON arrays of [lat, lon] pairs.
//
// kindLabel is mixed into the ETag so the 5 m and 50 m variants do not
// alias even when their underlying counts and updated_at coincide.
func (sv *server) handleListTrackPolylines(w http.ResponseWriter, r *http.Request, kind db.PreviewPolylineKind, kindLabel string) {
	ctx := r.Context()
	user := session.GetUser(ctx)

	params, err := parseListTracksParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := sv.d.ListTracksWithPolylines(ctx, params, kind, trackPolylinesLimit)
	if err != nil {
		logg.Error(ctx, "failed to list track polylines", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	var viewerID string
	if user != nil {
		viewerID = user.Uuid
	}
	eTag := fmt.Sprintf(`"%d-%d-%s-%s-v3"`,
		result.MaxUpdatedAt.UnixMilli(),
		result.TotalCount,
		viewerID,
		kindLabel,
	)
	if r.Header.Get(headerIfNoneMatch) == eTag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	entries := make([]trackPolylineEntry, len(result.Tracks))
	for i, t := range result.Tracks {
		pts, decErr := track.DecodeVarint(t.PolylineVarint)
		if decErr != nil {
			logg.Error(ctx, "decode preview polyline", "trackId", t.UUID, "err", decErr)
			pts = nil
		}
		latlon := make([][2]float64, len(pts))
		for j, p := range pts {
			latlon[j] = [2]float64{p.Lat, p.Lon}
		}
		entries[i] = trackPolylineEntry{
			UUID:           t.UUID,
			Name:           t.Name,
			UserID:         t.UserID,
			UserName:       t.UserName,
			TotalDistanceM: t.TotalDistanceM,
			TotalAscentM:   t.TotalAscentM,
			Bounds:         nullBBox(t.BoundsMinLat, t.BoundsMinLon, t.BoundsMaxLat, t.BoundsMaxLon),
			Polyline:       latlon,
			Starred:        t.Starred,
		}
		if t.Forecast.HasData() {
			entries[i].Forecast = &trackForecastResponse{
				ForecastReferenceTime: t.Forecast.ForecastReferenceTime.Time.Format(time.RFC3339),
				StartTime:             t.Forecast.StartTime.Time.Format(time.RFC3339),
				AvgTemperatureC:       nullFloat64Ptr(t.Forecast.AvgTemperatureC),
				TotalPrecipitationMm:  nullFloat64Ptr(t.Forecast.TotalPrecipitationMm),
				WindHeadMs:            nullFloat64Ptr(t.Forecast.WindHeadMs),
				WindRightMs:           nullFloat64Ptr(t.Forecast.WindRightMs),
				WindTailMs:            nullFloat64Ptr(t.Forecast.WindTailMs),
				WindLeftMs:            nullFloat64Ptr(t.Forecast.WindLeftMs),
			}
		}
	}

	w.Header().Set(headerCacheControl, "private, no-cache")
	w.Header().Set(headerETag, eTag)
	writeJSON(w, http.StatusOK, listTrackPolylinesResponse{
		Tracks:     entries,
		TotalCount: result.TotalCount,
		Limit:      trackPolylinesLimit,
	})
}

const maxBulkEditUUIDs = 500

type bulkEditTracksRequest struct {
	UUIDs         []string  `json:"uuids"`
	Public        *bool     `json:"public"`
	Source        *string   `json:"source"`
	Author        *string   `json:"author"`
	AuthorLinkURL *string   `json:"authorLinkUrl"`
	TrackType     *int64    `json:"trackType"`
	LinkURL       *string   `json:"linkUrl"`
	Sport         *int64    `json:"sport"`
	SubSport      *int64    `json:"subSport"`
	Tags          *[]string `json:"tags"`
}

func (sv *server) handleBulkEditTracks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.MustGetUser(ctx)

	var req bulkEditTracksRequest
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	if len(req.UUIDs) == 0 {
		writeError(w, http.StatusBadRequest, "uuids is required and must not be empty")
		return
	}
	if len(req.UUIDs) > maxBulkEditUUIDs {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("at most %d uuids allowed", maxBulkEditUUIDs))
		return
	}

	err := sv.d.BulkUpdateTracks(ctx, db.BulkUpdateTracksParams{
		UUIDs:         req.UUIDs,
		UserID:        user.Uuid,
		Public:        req.Public,
		Source:        req.Source,
		Author:        req.Author,
		AuthorLinkURL: req.AuthorLinkURL,
		TrackType:     req.TrackType,
		LinkURL:       req.LinkURL,
		Sport:         req.Sport,
		SubSport:      req.SubSport,
		Tags:          req.Tags,
	})
	if errors.Is(err, db.ErrBulkUpdateMismatch) {
		writeError(w, http.StatusNotFound, "one or more tracks not found or not owned by you")
		return
	}
	if err != nil {
		logg.Error(ctx, "failed to bulk update tracks", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (sv *server) handleListTracksForEditing(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.MustGetUser(ctx)

	tracks, err := sv.d.ListTracksForEditingForViewer(ctx, user.Uuid)
	if err != nil {
		logg.Error(ctx, "failed to list tracks for editing", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	trackUUIDs := make([]string, len(tracks))
	for i, t := range tracks {
		trackUUIDs[i] = t.Uuid
	}
	tagsByTrack, err := sv.d.GetTagsForTracks(ctx, trackUUIDs)
	if err != nil {
		logg.Error(ctx, "failed to get tags for editing tracks", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	responses := make([]trackResponse, len(tracks))
	for i, t := range tracks {
		tags := tagsByTrack[t.Uuid]
		if tags == nil {
			tags = []string{}
		}
		responses[i] = trackResponseFromDB(t, tags, nil, true)
	}

	writeJSON(w, http.StatusOK, map[string]any{"tracks": responses})
}

type editingCompleteRequest struct {
	UUIDs []string `json:"uuids"`
}

func (sv *server) handleEditingComplete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.MustGetUser(ctx)

	var req editingCompleteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	if len(req.UUIDs) == 0 {
		writeError(w, http.StatusBadRequest, "uuids is required and must not be empty")
		return
	}
	if len(req.UUIDs) > maxBulkEditUUIDs {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("at most %d uuids allowed", maxBulkEditUUIDs))
		return
	}

	err := sv.d.CompleteEditing(ctx, user.Uuid, req.UUIDs)
	if errors.Is(err, db.ErrBulkUpdateMismatch) {
		writeError(w, http.StatusNotFound, "one or more tracks not found or not owned by you")
		return
	}
	if err != nil {
		logg.Error(ctx, "failed to complete editing", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type trackStatisticsResponse struct {
	TotalDistanceM minMaxResponse `json:"totalDistanceM"`
	TotalAscentM   minMaxResponse `json:"totalAscentM"`
}

func (sv *server) handleTrackStatistics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.GetUser(ctx)

	statsParams := db.TrackStatisticsParams{}
	if user != nil {
		statsParams.UserID = user.Uuid
	}

	q := r.URL.Query()
	if v := q.Get("onlyMine"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid value for 'onlyMine'")
			return
		}
		if b && user != nil {
			statsParams.OnlyOwnedByUser = true
		}
	}

	result, err := sv.d.TrackStatistics(ctx, statsParams)
	if err != nil {
		logg.Error(ctx, "failed to get track statistics", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, trackStatisticsResponse{
		TotalDistanceM: minMaxResponse{
			Min: nullFloat64Ptr(result.TotalDistanceMMin),
			Max: nullFloat64Ptr(result.TotalDistanceMMax),
		},
		TotalAscentM: minMaxResponse{
			Min: nullFloat64Ptr(result.TotalAscentMMin),
			Max: nullFloat64Ptr(result.TotalAscentMMax),
		},
	})
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

const (
	maxUploadSize         = 5 << 20 // 5 MiB
	maxTrackPoints        = 100_000
	minTrackDistM         = 10       // 10 m
	maxTrackDistM         = 10_000e3 // 10 000 km
	maxTracksPerUser      = 200
	maxTracksPerUserAdmin = 10_000
)

var errUploadTrackLimitReached = errors.New("track limit reached")

// errUploadTrackDuplicate is returned when a user uploads a file whose content
// hash already exists in their library. It carries the UUID of the existing track.
type errUploadTrackDuplicate struct {
	existingUUID string
}

func (e *errUploadTrackDuplicate) Error() string {
	return fmt.Sprintf("duplicate track: %s", e.existingUUID)
}

func (sv *server) handleUploadTrack(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.MustGetUser(ctx)

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing or invalid file field")
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		logg.Error(ctx, "failed to read uploaded file", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	contentHash := sha256.Sum256(content)

	src, err := load.Blob(header.Filename, bytes.NewReader(content))
	if err != nil {
		if errors.Is(err, load.ErrUnsupportedFileExtension) {
			writeError(w, http.StatusBadRequest, "unsupported file type")
			return
		}
		logg.Error(ctx, "failed to parse track file", "filename", header.Filename, "err", err)
		writeError(w, http.StatusBadRequest, "failed to parse track file")
		return
	}

	t, err := track.New(src)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "failed to process track")
		return
	}
	if t.Len() > maxTrackPoints {
		writeError(w, http.StatusUnprocessableEntity, "track must have at most 100000 points")
		return
	}

	meta := t.EnhancedMetadata()

	// Compute the encoded preview polylines from the simplified point cloud.
	// This is cheap relative to the rest of the upload pipeline and avoids
	// having to re-load and re-parse the blob later. Any failure here aborts
	// the upload so a track row is never created without valid previews.
	pts := t.Points()
	previewDp5m, err := track.EncodeVarint(pts.SimplifyDP(track.PreviewPolylineEpsilon5M))
	if err != nil {
		logg.Error(ctx, "encode preview polyline 5m", "err", err)
		writeError(w, http.StatusUnprocessableEntity, "failed to encode track preview polyline")
		return
	}
	previewDp50m, err := track.EncodeVarint(pts.SimplifyDP(track.PreviewPolylineEpsilon50M))
	if err != nil {
		logg.Error(ctx, "encode preview polyline 50m", "err", err)
		writeError(w, http.StatusUnprocessableEntity, "failed to encode track preview polyline")
		return
	}

	meta.Name = strings.TrimSpace(meta.Name)
	if meta.Name == "" {
		name := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
		meta.Name = strings.TrimSpace(name)
	}
	if meta.Name == "" {
		meta.Name = "Track uploaded " + time.Now().UTC().Format("2006-01-02 15:04:05")
	}

	if meta.TotalDistanceM < minTrackDistM {
		writeError(w, http.StatusUnprocessableEntity, "track distance must be at least 10 m")
		return
	}
	if meta.TotalDistanceM > maxTrackDistM {
		writeError(w, http.StatusUnprocessableEntity, "track distance must be at most 10000 km")
		return
	}
	// meta.TotalDistanceM may be the device-reported value (FIT). Cross-check
	// the chord total computed from the points themselves so an upload whose
	// device total is plausible but whose points are clustered at the start
	// is also rejected.
	chordTotalM := pts[len(pts)-1].Distance
	if chordTotalM < minTrackDistM {
		writeError(w, http.StatusUnprocessableEntity, "track distance must be at least 10 m")
		return
	}
	if chordTotalM > maxTrackDistM {
		writeError(w, http.StatusUnprocessableEntity, "track distance must be at most 10000 km")
		return
	}

	trackID, err := uuid.NewV7()
	if err != nil {
		logg.Error(ctx, "failed to generate track uuid", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	now := time.Now().UTC()
	var created db.Track
	err = sv.d.WithTx(ctx, func(q *db.Queries) error {
		count, txErr := q.CountTracksByUser(ctx, user.Uuid)
		if txErr != nil {
			return txErr
		}
		limit := int64(maxTracksPerUser)
		if user.Admin != 0 {
			limit = maxTracksPerUserAdmin
		}
		if count >= limit {
			return errUploadTrackLimitReached
		}

		dupUUID, txErr := q.TrackExistsByUserAndBlobHash(ctx, db.TrackExistsByUserAndBlobHashParams{
			UserID: user.Uuid,
			Hash:   contentHash[:],
		})
		if txErr == nil {
			return &errUploadTrackDuplicate{existingUUID: dupUUID}
		}
		if !errors.Is(txErr, sql.ErrNoRows) {
			return txErr
		}

		// Reuse an existing blob if one with the same hash exists (cross-user dedup).
		var blobID int64
		existingID, txErr := q.GetBlobIDByHash(ctx, db.GetBlobIDByHashParams{
			HashType: int64(blob.HashTypeSHA256),
			Hash:     contentHash[:],
		})
		if txErr == nil {
			blobID = existingID
		} else if errors.Is(txErr, sql.ErrNoRows) {
			b, blobErr := blob.Create(ctx, q, content, blob.CompressionZstd)
			if blobErr != nil {
				return blobErr
			}
			blobID = b.ID
		} else {
			return txErr
		}

		created, txErr = q.CreateTrack(ctx, db.CreateTrackParams{
			Uuid:                trackID.String(),
			CreatedAt:           now,
			UpdatedAt:           now,
			UserID:              user.Uuid,
			BlobID:              blobID,
			FileFormat:          int64(fileFormatFromExt(header.Filename)),
			OriginalFilename:    header.Filename,
			Name:                meta.Name,
			Description:         toNullString(meta.Description),
			Source:              toNullString(meta.Source),
			Author:              toNullString(meta.Author),
			AuthorLinkUrl:       toNullString(meta.AuthorLinkURL),
			TrackType:           int64(meta.TrackType),
			LinkUrl:             toNullString(meta.LinkURL),
			Sport:               int64(meta.Sport),
			SubSport:            int64(meta.SubSport),
			TotalDistanceM:      meta.TotalDistanceM,
			TotalAscentM:        meta.TotalAscentM,
			StartLat:            toNullFloat64(meta.StartLat),
			StartLon:            toNullFloat64(meta.StartLon),
			EndLat:              toNullFloat64(meta.EndLat),
			EndLon:              toNullFloat64(meta.EndLon),
			BoundsMinLat:        toNullFloat64(meta.BoundsMinLat),
			BoundsMinLon:        toNullFloat64(meta.BoundsMinLon),
			BoundsMaxLat:        toNullFloat64(meta.BoundsMaxLat),
			BoundsMaxLon:        toNullFloat64(meta.BoundsMaxLon),
			MinElevationM:       toNullFloat64(meta.MinElevationM),
			MaxElevationM:       toNullFloat64(meta.MaxElevationM),
			OriginalCreatedAt:   toNullTime(meta.OriginalCreatedAt),
			Public:              0,
			PolylineDp5mVarint:  previewDp5m,
			PolylineDp50mVarint: previewDp50m,
		})
		return txErr
	})
	if errors.Is(err, errUploadTrackLimitReached) {
		writeError(w, http.StatusConflict, "per-user track limit reached")
		return
	}
	var dupErr *errUploadTrackDuplicate
	if errors.As(err, &dupErr) {
		writeError(w, http.StatusConflict, fmt.Sprintf("duplicate file (%s)", dupErr.existingUUID))
		return
	}
	if err != nil {
		logg.Error(ctx, "failed to store track", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	// Schedule geoname labeling in the background.
	if submitErr := jobs.Submit(ctx, sv.jobSubmitter, geonames.LabelerArgs{
		TrackID: created.Uuid,
	}, jobs.Params{MaxRetries: 2}); submitErr != nil {
		logg.Error(ctx, "failed to submit labeler job", "err", submitErr)
	}

	// Schedule forecast computation for the new track.
	if submitErr := jobs.Submit(ctx, sv.jobSubmitter, forecast.SummarizerArgs{
		TrackUUID: created.Uuid,
	}, jobs.Params{MaxRetries: 2}); submitErr != nil {
		logg.Error(ctx, "failed to submit forecast summarizer job", "err", submitErr)
	}

	// Schedule track grouping with debounce so rapid uploads are coalesced.
	if submitErr := jobs.Submit(ctx, sv.jobSubmitter, trackgroup.GrouperArgs{
		UserID: user.Uuid,
	}, jobs.Params{DelayS: 1 * time.Minute, Debounce: true}); submitErr != nil {
		logg.Error(ctx, "failed to submit grouper job", "err", submitErr)
	}

	// TODO: Uncomment this again as soon as segmenting works.
	// // Schedule segment extraction with debounce so rapid uploads are coalesced.
	// if submitErr := jobs.Submit(ctx, sv.jobSubmitter, segment.BuilderArgs{},
	// 	jobs.Params{DelayS: 2 * time.Minute, Debounce: true}); submitErr != nil {
	// 	logg.Error(ctx, "failed to submit segment builder job", "err", submitErr)
	// }

	writeJSON(w, http.StatusCreated, trackResponseFromDB(db.TrackWithStarred{
		Track:          created,
		UserName:       user.Name,
		UserAvatarSeed: user.AvatarSeed,
	}, nil, nil, true))
}
