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

	"github.com/google/uuid"
	"jo-m.ch/go/detour/internal/pkg/blob"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/load"
	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/session"
	"jo-m.ch/go/detour/internal/pkg/track"
)

type trackResponse struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Description    string  `json:"description,omitempty"`
	Source         string  `json:"source,omitempty"`
	Author         string  `json:"author,omitempty"`
	AuthorLinkURL  string  `json:"authorLinkUrl,omitempty"`
	FileFormat     int     `json:"fileFormat"`
	TrackType      int     `json:"trackType"`
	LinkURL        string  `json:"linkUrl,omitempty"`
	Sport          int     `json:"sport"`
	SubSport       int     `json:"subSport"`
	TotalDistanceM float64 `json:"totalDistanceM"`
	TotalAscentM   float64 `json:"totalAscentM"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt"`
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{Valid: true, String: s}
}

func nullStringVal(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func trackResponseFromDB(t db.Track) trackResponse {
	return trackResponse{
		ID:             t.ID,
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
		CreatedAt:      t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      t.UpdatedAt.Format(time.RFC3339),
	}
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
			ID:             trackID.String(),
			CreatedAt:      now,
			UpdatedAt:      now,
			UserID:         user.ID,
			BlobID:         blobID.String(),
			FileFormat:     int64(fileFormatFromExt(header.Filename)),
			Name:           meta.Name,
			Description:    toNullString(meta.Description),
			Source:         toNullString(meta.Source),
			Author:         toNullString(meta.Author),
			AuthorLinkUrl:  toNullString(meta.AuthorLinkURL),
			TrackType:      int64(meta.TrackType),
			LinkUrl:        toNullString(meta.LinkURL),
			Sport:          int64(meta.Sport),
			SubSport:       int64(meta.SubSport),
			TotalDistanceM: meta.TotalDistanceM,
			TotalAscentM:   meta.TotalAscentM,
		})
		return err
	})
	if err != nil {
		logg.Error(ctx, "failed to store track", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, trackResponseFromDB(created))
}
