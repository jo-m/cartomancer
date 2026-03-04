package rest

import (
	"bytes"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"path/filepath"
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
	UUID              string   `json:"uuid"`
	Name              string   `json:"name"`
	Description       string   `json:"description,omitempty"`
	Source            string   `json:"source,omitempty"`
	Author            string   `json:"author,omitempty"`
	AuthorLinkURL     string   `json:"authorLinkUrl,omitempty"`
	FileFormat        int      `json:"fileFormat"`
	TrackType         int      `json:"trackType"`
	LinkURL           string   `json:"linkUrl,omitempty"`
	Sport             int      `json:"sport"`
	SubSport          int      `json:"subSport"`
	TotalDistanceM    float64  `json:"totalDistanceM"`
	TotalAscentM      float64  `json:"totalAscentM"`
	StartLat          *float64 `json:"startLat,omitempty"`
	StartLon          *float64 `json:"startLon,omitempty"`
	EndLat            *float64 `json:"endLat,omitempty"`
	EndLon            *float64 `json:"endLon,omitempty"`
	OriginalCreatedAt string   `json:"originalCreatedAt,omitempty"`
	CreatedAt         string   `json:"createdAt"`
	UpdatedAt         string   `json:"updatedAt"`
	Public            bool     `json:"public"`
	Tags              []string `json:"tags"`
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
		UUID:           t.Uuid,
		Name:           t.Name,
		Description:    nullStringVal(t.Description),
		Source:         nullStringVal(t.Source),
		Author:         nullStringVal(t.Author),
		AuthorLinkURL:  nullStringVal(t.AuthorLinkUrl),
		FileFormat:     int(t.FileFormat),
		TrackType:      int(t.TrackType),
		LinkURL:        nullStringVal(t.LinkUrl),
		Sport:          int(t.Sport),
		SubSport:       int(t.SubSport),
		TotalDistanceM: t.TotalDistanceM,
		TotalAscentM:   t.TotalAscentM,
		StartLat:       nullFloat64Ptr(t.StartLat),
		StartLon:       nullFloat64Ptr(t.StartLon),
		EndLat:         nullFloat64Ptr(t.EndLat),
		EndLon:         nullFloat64Ptr(t.EndLon),
		CreatedAt:      t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      t.UpdatedAt.Format(time.RFC3339),
		Public:         t.Public != 0,
		Tags:           tags,
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

	user := session.GetUser(ctx)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	trackUUID := chi.URLParam(r, "uuid")

	var req editTrackRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
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

	existing, err := sv.d.QueryRO().GetTrackByUUID(ctx, trackUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "track not found")
			return
		}
		logg.Error(ctx, "failed to get track", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if existing.UserID != user.Uuid {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	now := time.Now().UTC()
	var updated db.Track
	err = sv.d.WithTx(ctx, func(q *db.Queries) error {
		var txErr error
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
			t, txErr := q.UpsertTag(ctx, tag)
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
	if err != nil {
		logg.Error(ctx, "failed to update track", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, trackResponseFromDB(updated, req.Tags))
}

// TODO: Make configurable.
const maxUploadSize = 50 << 20 // 50 MB

func (sv *server) handleUploadTrack(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := session.GetUser(ctx)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

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
		writeError(w, http.StatusInternalServerError, "internal server error")
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
	if t.Len() < 3 {
		writeError(w, http.StatusUnprocessableEntity, "track must have at least 3 points")
		return
	}

	meta := t.EnhancedMetadata()

	blobID, err := uuid.NewV7()
	if err != nil {
		logg.Error(ctx, "failed to generate blob uuid", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	trackID, err := uuid.NewV7()
	if err != nil {
		logg.Error(ctx, "failed to generate track uuid", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	now := time.Now().UTC()
	var created db.Track
	err = sv.d.WithTx(ctx, func(q *db.Queries) error {
		_, err := blob.Create(ctx, q, blobID.String(), header.Filename, content, blob.CompressionZstd)
		if err != nil {
			return err
		}

		created, err = q.CreateTrack(ctx, db.CreateTrackParams{
			Uuid:              trackID.String(),
			CreatedAt:         now,
			UpdatedAt:         now,
			UserID:            user.Uuid,
			BlobID:            blobID.String(),
			FileFormat:        int64(fileFormatFromExt(header.Filename)),
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
		})
		return err
	})
	if err != nil {
		logg.Error(ctx, "failed to store track", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, trackResponseFromDB(created, nil))
}
