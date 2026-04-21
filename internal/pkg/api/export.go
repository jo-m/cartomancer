package api

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"jo-m.ch/go/cartomancer/internal/pkg/blob"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/session"
	"jo-m.ch/go/cartomancer/internal/pkg/track"
)

// exportTrackMeta is the per-track metadata included in the export.
type exportTrackMeta struct {
	UUID              string   `json:"uuid"`
	Name              string   `json:"name"`
	Description       string   `json:"description,omitempty"`
	Source            string   `json:"source,omitempty"`
	Author            string   `json:"author,omitempty"`
	AuthorLinkURL     string   `json:"authorLinkUrl,omitempty"`
	FileFormat        string   `json:"fileFormat"`
	TrackType         string   `json:"trackType"`
	LinkURL           string   `json:"linkUrl,omitempty"`
	Sport             string   `json:"sport"`
	SubSport          string   `json:"subSport"`
	TotalDistanceM    float64  `json:"totalDistanceM"`
	TotalAscentM      float64  `json:"totalAscentM"`
	MinElevationM     *float64 `json:"minElevationM,omitempty"`
	MaxElevationM     *float64 `json:"maxElevationM,omitempty"`
	StartLat          *float64 `json:"startLat,omitempty"`
	StartLon          *float64 `json:"startLon,omitempty"`
	EndLat            *float64 `json:"endLat,omitempty"`
	EndLon            *float64 `json:"endLon,omitempty"`
	Public            bool     `json:"public"`
	Tags              []string `json:"tags"`
	OriginalFilename  string   `json:"originalFilename"`
	Filename          string   `json:"filename"`
	CreatedAt         string   `json:"createdAt"`
	UpdatedAt         string   `json:"updatedAt"`
	OriginalCreatedAt string   `json:"originalCreatedAt,omitempty"`
}

// exportUserMeta holds the user profile data included in the export.
type exportUserMeta struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// exportTrackGroup represents a group of similar tracks.
type exportTrackGroup struct {
	UUID     string   `json:"uuid"`
	TrackIDs []string `json:"trackIds"`
}

// exportBlobError records a track blob that could not be included in the export.
type exportBlobError struct {
	TrackUUID string `json:"trackUuid"`
	Error     string `json:"error"`
}

func fileFormatLabel(f int64) string {
	switch track.FileFormat(f) {
	case track.FileFormatGPX:
		return "gpx"
	case track.FileFormatFIT:
		return "fit"
	default:
		return "unknown"
	}
}

func trackTypeLabel(t int64) string {
	switch track.TrackType(t) {
	case track.TrackTypePlanned:
		return "planned"
	case track.TrackTypeRecorded:
		return "recorded"
	default:
		return "unknown"
	}
}

func sportLabel(s int64) string {
	switch track.Sport(s) {
	case track.SportRunning:
		return "running"
	case track.SportCycling:
		return "cycling"
	default:
		return "unknown"
	}
}

func subSportLabel(s int64) string {
	switch track.SubSport(s) {
	case track.SubSportRunningOutdoor:
		return "running_outdoor"
	case track.SubSportRunningTreadmill:
		return "running_treadmill"
	case track.SubSportCyclingRoad:
		return "cycling_road"
	case track.SubSportCyclingSpinning:
		return "cycling_spinning"
	case track.SubSportCyclingIndoorCycling:
		return "cycling_indoor"
	case track.SubSportCyclingMountain:
		return "cycling_mountain"
	case track.SubSportCyclingGravel:
		return "cycling_gravel"
	case track.SubSportCyclingCommuting:
		return "cycling_commuting"
	default:
		return "unknown"
	}
}

func fileFormatExt(f int64) string {
	switch track.FileFormat(f) {
	case track.FileFormatGPX:
		return ".gpx"
	case track.FileFormatFIT:
		return ".fit"
	default:
		return ".bin"
	}
}

// writeZipJSON creates a JSON file inside the zip archive.
func writeZipJSON(zw *zip.Writer, name string, modified time.Time, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", name, err)
	}
	w, err := zw.CreateHeader(&zip.FileHeader{
		Name:     name,
		Method:   zip.Deflate,
		Modified: modified,
	})
	if err != nil {
		return fmt.Errorf("create %s: %w", name, err)
	}
	_, err = w.Write(data)
	return err
}

// handleExportData streams a ZIP archive containing all of the authenticated
// user's tracks, metadata, and track groups. Generated data (forecasts,
// location summaries) is excluded per GDPR data-portability requirements.
func (sv *server) handleExportData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.MustGetUser(ctx)

	u, err := sv.d.QueryRO().GetUser(ctx, user.Uuid)
	if err != nil {
		logg.Error(ctx, "export: failed to get user", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	tracks, err := sv.d.QueryRO().ListTracksByUser(ctx, user.Uuid)
	if err != nil {
		logg.Error(ctx, "export: failed to list tracks", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	trackUUIDs := make([]string, len(tracks))
	for i, t := range tracks {
		trackUUIDs[i] = t.Uuid
	}

	tagsMap, err := sv.d.GetTagsForTracks(ctx, trackUUIDs)
	if err != nil {
		logg.Error(ctx, "export: failed to get tags", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	groupRows, err := sv.d.QueryRO().ListTrackGroupsByUser(ctx, user.Uuid)
	if err != nil {
		logg.Error(ctx, "export: failed to list track groups", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	// Aggregate flat group rows into grouped structure.
	groupOrder := []string{}
	groupMap := map[string]*exportTrackGroup{}
	for _, row := range groupRows {
		g, ok := groupMap[row.Uuid]
		if !ok {
			g = &exportTrackGroup{UUID: row.Uuid}
			groupMap[row.Uuid] = g
			groupOrder = append(groupOrder, row.Uuid)
		}
		g.TrackIDs = append(g.TrackIDs, row.TrackID)
	}
	groups := make([]exportTrackGroup, len(groupOrder))
	for i, id := range groupOrder {
		groups[i] = *groupMap[id]
	}

	now := time.Now().UTC()

	trackMetas := make([]exportTrackMeta, len(tracks))
	for i, t := range tracks {
		tags := tagsMap[t.Uuid]
		if tags == nil {
			tags = []string{}
		}

		ext := fileFormatExt(t.FileFormat)
		filename := fmt.Sprintf("tracks/%s%s", t.Uuid, ext)

		tm := exportTrackMeta{
			UUID:             t.Uuid,
			Name:             t.Name,
			Description:      nullStringVal(t.Description),
			Source:           nullStringVal(t.Source),
			Author:           nullStringVal(t.Author),
			AuthorLinkURL:    nullStringVal(t.AuthorLinkUrl),
			FileFormat:       fileFormatLabel(t.FileFormat),
			TrackType:        trackTypeLabel(t.TrackType),
			LinkURL:          nullStringVal(t.LinkUrl),
			Sport:            sportLabel(t.Sport),
			SubSport:         subSportLabel(t.SubSport),
			TotalDistanceM:   t.TotalDistanceM,
			TotalAscentM:     t.TotalAscentM,
			MinElevationM:    nullFloat64Ptr(t.MinElevationM),
			MaxElevationM:    nullFloat64Ptr(t.MaxElevationM),
			StartLat:         nullFloat64Ptr(t.StartLat),
			StartLon:         nullFloat64Ptr(t.StartLon),
			EndLat:           nullFloat64Ptr(t.EndLat),
			EndLon:           nullFloat64Ptr(t.EndLon),
			Public:           t.Public != 0,
			Tags:             tags,
			OriginalFilename: t.OriginalFilename,
			Filename:         filename,
			CreatedAt:        t.CreatedAt.Format(time.RFC3339),
			UpdatedAt:        t.UpdatedAt.Format(time.RFC3339),
		}
		if t.OriginalCreatedAt.Valid {
			tm.OriginalCreatedAt = t.OriginalCreatedAt.Time.Format(time.RFC3339)
		}

		trackMetas[i] = tm
	}

	archiveName := fmt.Sprintf("cartomancer-export-%s.zip", now.Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, archiveName))

	zw := zip.NewWriter(w)
	defer zw.Close()

	userMeta := exportUserMeta{
		UUID:      u.Uuid,
		Name:      u.Name,
		Email:     u.Email,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
		UpdatedAt: u.UpdatedAt.Format(time.RFC3339),
	}
	if err := writeZipJSON(zw, "user.json", now, userMeta); err != nil {
		logg.Error(ctx, "export: failed to write user.json", "err", err)
		return
	}

	if err := writeZipJSON(zw, "tracks.json", now, trackMetas); err != nil {
		logg.Error(ctx, "export: failed to write tracks.json", "err", err)
		return
	}

	if err := writeZipJSON(zw, "track_groups.json", now, groups); err != nil {
		logg.Error(ctx, "export: failed to write track_groups.json", "err", err)
		return
	}

	var blobErrors []exportBlobError
	for _, t := range tracks {
		b, blobErr := blob.Get(ctx, sv.d.QueryRO(), t.BlobID)
		if blobErr != nil {
			logg.Error(ctx, "export: failed to get blob", "err", blobErr, "track", t.Uuid)
			blobErrors = append(blobErrors, exportBlobError{TrackUUID: t.Uuid, Error: blobErr.Error()})
			continue
		}

		ext := fileFormatExt(t.FileFormat)
		entryName := fmt.Sprintf("tracks/%s%s", t.Uuid, ext)

		fw, fwErr := zw.CreateHeader(&zip.FileHeader{
			Name:     entryName,
			Method:   zip.Deflate,
			Modified: t.CreatedAt,
		})
		if fwErr != nil {
			logg.Error(ctx, "export: failed to create zip entry", "err", fwErr, "track", t.Uuid)
			blobErrors = append(blobErrors, exportBlobError{TrackUUID: t.Uuid, Error: fwErr.Error()})
			continue
		}
		if _, fwErr = fw.Write(b.Content); fwErr != nil {
			logg.Error(ctx, "export: failed to write track file", "err", fwErr, "track", t.Uuid)
			return
		}
	}

	if len(blobErrors) > 0 {
		if err := writeZipJSON(zw, "errors.json", now, blobErrors); err != nil {
			logg.Error(ctx, "export: failed to write errors.json", "err", err)
		}
	}
}
