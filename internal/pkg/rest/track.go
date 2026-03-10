package rest

import (
	"bytes"
	"database/sql"
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
	"jo-m.ch/go/detour/internal/pkg/blob"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/load"
	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/session"
	"jo-m.ch/go/detour/internal/pkg/track"
)

type trackResponse struct {
	UUID                    string   `json:"uuid"`
	Name                    string   `json:"name"`
	Description             string   `json:"description,omitempty"`
	Source                  string   `json:"source,omitempty"`
	Author                  string   `json:"author,omitempty"`
	AuthorLinkURL           string   `json:"authorLinkUrl,omitempty"`
	FileFormat              int      `json:"fileFormat"`
	TrackType               int      `json:"trackType"`
	LinkURL                 string   `json:"linkUrl,omitempty"`
	Sport                   int      `json:"sport"`
	SubSport                int      `json:"subSport"`
	TotalDistanceM          float64  `json:"totalDistanceM"`
	TotalAscentM            float64  `json:"totalAscentM"`
	StartLat                *float64 `json:"startLat,omitempty"`
	StartLon                *float64 `json:"startLon,omitempty"`
	EndLat                  *float64 `json:"endLat,omitempty"`
	EndLon                  *float64 `json:"endLon,omitempty"`
	OriginalCreatedAt       string   `json:"originalCreatedAt,omitempty"`
	CreatedAt               string   `json:"createdAt"`
	UpdatedAt               string   `json:"updatedAt"`
	Public                  bool     `json:"public"`
	InitialEditingCompleted bool     `json:"initialEditingCompleted"`
	Tags                    []string `json:"tags"`
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

func trackResponseFromDB(t db.Track, tags []string) trackResponse {
	if tags == nil {
		tags = []string{}
	}
	resp := trackResponse{
		UUID:                    t.Uuid,
		Name:                    t.Name,
		Description:             nullStringVal(t.Description),
		Source:                  nullStringVal(t.Source),
		Author:                  nullStringVal(t.Author),
		AuthorLinkURL:           nullStringVal(t.AuthorLinkUrl),
		FileFormat:              int(t.FileFormat),
		TrackType:               int(t.TrackType),
		LinkURL:                 nullStringVal(t.LinkUrl),
		Sport:                   int(t.Sport),
		SubSport:                int(t.SubSport),
		TotalDistanceM:          t.TotalDistanceM,
		TotalAscentM:            t.TotalAscentM,
		StartLat:                nullFloat64Ptr(t.StartLat),
		StartLon:                nullFloat64Ptr(t.StartLon),
		EndLat:                  nullFloat64Ptr(t.EndLat),
		EndLon:                  nullFloat64Ptr(t.EndLon),
		CreatedAt:               t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:               t.UpdatedAt.Format(time.RFC3339),
		Public:                  t.Public != 0,
		InitialEditingCompleted: t.InitialEditingCompleted != 0,
		Tags:                    tags,
	}
	if t.OriginalCreatedAt.Valid {
		resp.OriginalCreatedAt = t.OriginalCreatedAt.Time.Format(time.RFC3339)
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

func (sv *server) handleGetTrack(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.MustGetUser(ctx)
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

	if t.Public == 0 && user.Uuid != t.UserID {
		writeError(w, http.StatusNotFound, "track not found")
		return
	}

	tags, err := sv.d.QueryRO().GetTagsByTrackID(ctx, trackUUID)
	if err != nil {
		logg.Error(ctx, "failed to get track tags", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, trackResponseFromDB(t, tags))
}

func (sv *server) handleDownloadTrackBlob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.MustGetUser(ctx)
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

	if t.Public == 0 && user.Uuid != t.UserID {
		writeError(w, http.StatusNotFound, "track not found")
		return
	}

	var contentType string
	switch track.FileFormat(t.FileFormat) {
	case track.FileFormatGPX:
		contentType = "application/gpx+xml"
	case track.FileFormatFIT:
		contentType = "application/vnd.ant.fit"
	default:
		contentType = "application/octet-stream"
	}

	if err := blob.Serve(w, r, sv.d.QueryRO(), t.BlobID, contentType, t.OriginalFilename); err != nil {
		logg.Error(ctx, "failed to serve blob", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}
}

func (sv *server) handleDownloadTrackSVG(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.MustGetUser(ctx)
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

	if t.Public == 0 && user.Uuid != t.UserID {
		writeError(w, http.StatusNotFound, "track not found")
		return
	}

	if !t.PreviewSvgBlobID.Valid {
		writeError(w, http.StatusNotFound, "preview not available")
		return
	}

	if err := blob.Serve(w, r, sv.d.QueryRO(), t.PreviewSvgBlobID.String, "image/svg+xml", t.Uuid+".svg"); err != nil {
		logg.Error(ctx, "failed to serve preview svg", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}
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
			writeError(w, http.StatusBadRequest, "each tag must be 2-32 characters")
			return
		}
	}

	var errForbidden = errors.New("forbidden")

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

	writeJSON(w, http.StatusOK, trackResponseFromDB(updated, req.Tags))
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

func (sv *server) handleListTracks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.MustGetUser(ctx)

	q := r.URL.Query()
	qmap := map[string][]string(q)

	params := db.ListTracksParams{
		UserID: user.Uuid,
	}

	// Pagination.
	if v := q.Get("page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "invalid value for 'page'")
			return
		}
		params.Page = n
	}
	if v := q.Get("pageSize"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 200 {
			writeError(w, http.StatusBadRequest, "invalid value for 'pageSize': must be 1-200")
			return
		}
		params.PageSize = n
	}

	// Public filter.
	if v := q.Get("public"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid value for 'public'")
			return
		}
		params.Public = &b
	}

	// Enum multi-value filters.
	var err error
	if params.FileFormats, err = parseInt64Slice(qmap, "fileFormat"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if params.TrackTypes, err = parseInt64Slice(qmap, "trackType"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if params.Sports, err = parseInt64Slice(qmap, "sport"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if params.SubSports, err = parseInt64Slice(qmap, "subSport"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Text filters.
	params.Name = parseOptionalString(qmap, "name")
	params.Description = parseOptionalString(qmap, "description")
	params.Source = parseOptionalString(qmap, "source")

	// Datetime range filters.
	if params.CreatedAtMin, err = parseOptionalTime(qmap, "createdAtMin"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if params.CreatedAtMax, err = parseOptionalTime(qmap, "createdAtMax"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if params.UpdatedAtMin, err = parseOptionalTime(qmap, "updatedAtMin"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if params.UpdatedAtMax, err = parseOptionalTime(qmap, "updatedAtMax"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if params.OriginalCreatedAtMin, err = parseOptionalTime(qmap, "originalCreatedAtMin"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if params.OriginalCreatedAtMax, err = parseOptionalTime(qmap, "originalCreatedAtMax"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Numeric range filters.
	if params.TotalDistanceMMin, err = parseOptionalFloat64(qmap, "totalDistanceMMin"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if params.TotalDistanceMMax, err = parseOptionalFloat64(qmap, "totalDistanceMMax"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if params.TotalAscentMMin, err = parseOptionalFloat64(qmap, "totalAscentMMin"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if params.TotalAscentMMax, err = parseOptionalFloat64(qmap, "totalAscentMMax"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Coordinate bounding box filters.
	if params.StartLatMin, err = parseOptionalFloat64(qmap, "startLatMin"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if params.StartLatMax, err = parseOptionalFloat64(qmap, "startLatMax"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if params.StartLonMin, err = parseOptionalFloat64(qmap, "startLonMin"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if params.StartLonMax, err = parseOptionalFloat64(qmap, "startLonMax"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if params.EndLatMin, err = parseOptionalFloat64(qmap, "endLatMin"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if params.EndLatMax, err = parseOptionalFloat64(qmap, "endLatMax"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if params.EndLonMin, err = parseOptionalFloat64(qmap, "endLonMin"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if params.EndLonMax, err = parseOptionalFloat64(qmap, "endLonMax"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Radial filters.
	if params.StartNearLat, err = parseOptionalFloat64(qmap, "startNearLat"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if params.StartNearLon, err = parseOptionalFloat64(qmap, "startNearLon"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if params.StartNearRadiusM, err = parseOptionalFloat64(qmap, "startNearRadiusM"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if params.EndNearLat, err = parseOptionalFloat64(qmap, "endNearLat"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if params.EndNearLon, err = parseOptionalFloat64(qmap, "endNearLon"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if params.EndNearRadiusM, err = parseOptionalFloat64(qmap, "endNearRadiusM"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Validate radial filters: all three or none.
	startNearCount := boolToInt(params.StartNearLat != nil) + boolToInt(params.StartNearLon != nil) + boolToInt(params.StartNearRadiusM != nil)
	if startNearCount > 0 && startNearCount < 3 {
		writeError(w, http.StatusBadRequest, "startNearLat, startNearLon, and startNearRadiusM must all be provided together")
		return
	}
	endNearCount := boolToInt(params.EndNearLat != nil) + boolToInt(params.EndNearLon != nil) + boolToInt(params.EndNearRadiusM != nil)
	if endNearCount > 0 && endNearCount < 3 {
		writeError(w, http.StatusBadRequest, "endNearLat, endNearLon, and endNearRadiusM must all be provided together")
		return
	}

	result, err := sv.d.ListTracks(ctx, params)
	if err != nil {
		logg.Error(ctx, "failed to list tracks", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	// Fetch tags for all returned tracks in a single query.
	trackUUIDs := make([]string, len(result.Tracks))
	for i, t := range result.Tracks {
		trackUUIDs[i] = t.Uuid
	}
	tagsByTrack, err := sv.d.GetTagsForTracks(ctx, trackUUIDs)
	if err != nil {
		logg.Error(ctx, "failed to get tags for tracks", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	responses := make([]trackResponse, len(result.Tracks))
	for i, t := range result.Tracks {
		tags := tagsByTrack[t.Uuid]
		if tags == nil {
			tags = []string{}
		}
		responses[i] = trackResponseFromDB(t, tags)
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

const maxBulkEditUUIDs = 10000

type bulkEditTracksRequest struct {
	UUIDs         []string `json:"uuids"`
	Public        *bool    `json:"public"`
	Source        *string  `json:"source"`
	Author        *string  `json:"author"`
	AuthorLinkURL *string  `json:"authorLinkUrl"`
	TrackType     *int64   `json:"trackType"`
	LinkURL       *string  `json:"linkUrl"`
	Sport         *int64   `json:"sport"`
	SubSport      *int64   `json:"subSport"`
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

	tracks, err := sv.d.QueryRO().ListTracksForEditing(ctx, user.Uuid)
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
		responses[i] = trackResponseFromDB(t, tags)
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
	TotalDistanceMMin *float64 `json:"totalDistanceMMin"`
	TotalDistanceMMax *float64 `json:"totalDistanceMMax"`
	TotalAscentMMin   *float64 `json:"totalAscentMMin"`
	TotalAscentMMax   *float64 `json:"totalAscentMMax"`
}

func (sv *server) handleTrackStatistics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.MustGetUser(ctx)

	result, err := sv.d.TrackStatistics(ctx, db.TrackStatisticsParams{
		UserID: user.Uuid,
	})
	if err != nil {
		logg.Error(ctx, "failed to get track statistics", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, trackStatisticsResponse{
		TotalDistanceMMin: nullFloat64Ptr(result.TotalDistanceMMin),
		TotalDistanceMMax: nullFloat64Ptr(result.TotalDistanceMMax),
		TotalAscentMMin:   nullFloat64Ptr(result.TotalAscentMMin),
		TotalAscentMMax:   nullFloat64Ptr(result.TotalAscentMMax),
	})
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

const (
	maxUploadSize    = 5 << 20 // 5 MiB
	minTrackPoints   = 3
	maxTrackPoints   = 100_000
	minTrackDistM    = 10       // 10 m
	maxTrackDistM    = 10_000e3 // 10 000 km
	maxTracksPerUser = 10_000
)

var errUploadTrackLimitReached = errors.New("track limit reached")

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

	// TODO: Not necessary to buffer to memory here.
	content, err := io.ReadAll(file)
	if err != nil {
		logg.Error(ctx, "failed to read uploaded file", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

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

	t := track.New(src, 0)
	if t.Len() < minTrackPoints {
		writeError(w, http.StatusUnprocessableEntity, "track must have at least 3 points")
		return
	}
	if t.Len() > maxTrackPoints {
		writeError(w, http.StatusUnprocessableEntity, "track must have at most 100000 points")
		return
	}

	meta := t.EnhancedMetadata()

	if meta.TotalDistanceM < minTrackDistM {
		writeError(w, http.StatusUnprocessableEntity, "track distance must be at least 10 m")
		return
	}
	if meta.TotalDistanceM > maxTrackDistM {
		writeError(w, http.StatusUnprocessableEntity, "track distance must be at most 10000 km")
		return
	}

	blobID, err := uuid.NewV7()
	if err != nil {
		logg.Error(ctx, "failed to generate blob uuid", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	svgBlobID, err := uuid.NewV7()
	if err != nil {
		logg.Error(ctx, "failed to generate svg blob uuid", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	trackID, err := uuid.NewV7()
	if err != nil {
		logg.Error(ctx, "failed to generate track uuid", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	svgContent := []byte(t.PreviewSVG(512))

	now := time.Now().UTC()
	var created db.Track
	err = sv.d.WithTx(ctx, func(q *db.Queries) error {
		count, txErr := q.CountTracksByUser(ctx, user.Uuid)
		if txErr != nil {
			return txErr
		}
		if count >= maxTracksPerUser {
			return errUploadTrackLimitReached
		}

		_, txErr = blob.Create(ctx, q, blobID.String(), content, blob.CompressionZstd)
		if txErr != nil {
			return txErr
		}

		_, txErr = blob.Create(ctx, q, svgBlobID.String(), svgContent, blob.CompressionZstd)
		if txErr != nil {
			return txErr
		}

		created, txErr = q.CreateTrack(ctx, db.CreateTrackParams{
			Uuid:              trackID.String(),
			CreatedAt:         now,
			UpdatedAt:         now,
			UserID:            user.Uuid,
			BlobID:            blobID.String(),
			FileFormat:        int64(fileFormatFromExt(header.Filename)),
			OriginalFilename:  header.Filename,
			Name:              meta.Name,
			Description:       toNullString(meta.Description),
			Source:            toNullString(meta.Source),
			Author:            toNullString(meta.Author),
			AuthorLinkUrl:     toNullString(meta.AuthorLinkURL),
			TrackType:         int64(meta.TrackType),
			LinkUrl:           toNullString(meta.LinkURL),
			Sport:             int64(meta.Sport),
			SubSport:          int64(meta.SubSport),
			TotalDistanceM:    meta.TotalDistanceM,
			TotalAscentM:      meta.TotalAscentM,
			StartLat:          toNullFloat64(meta.StartLat),
			StartLon:          toNullFloat64(meta.StartLon),
			EndLat:            toNullFloat64(meta.EndLat),
			EndLon:            toNullFloat64(meta.EndLon),
			OriginalCreatedAt: toNullTime(meta.OriginalCreatedAt),
			Public:            0,
			PreviewSvgBlobID:  sql.NullString{Valid: true, String: svgBlobID.String()},
		})
		return txErr
	})
	if errors.Is(err, errUploadTrackLimitReached) {
		writeError(w, http.StatusConflict, "track limit reached (max 10000)")
		return
	}
	if err != nil {
		logg.Error(ctx, "failed to store track", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, trackResponseFromDB(created, nil))
}
