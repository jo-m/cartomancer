package rest

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/session"
)

// handleStarTrack handles POST /tracks/{uuid}/star.
// Authenticated users may star any track visible to them.
// Starring a track that is already starred is idempotent.
func (sv *server) handleStarTrack(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.MustGetUser(ctx)
	trackUUID := chi.URLParam(r, "uuid")

	t, err := sv.d.QueryRO().GetTrackByUUID(ctx, trackUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "track not found")
			return
		}
		logg.Error(ctx, "failed to get track for star", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	if t.Public == 0 && t.UserID != user.Uuid {
		writeError(w, http.StatusNotFound, "track not found")
		return
	}

	if err := sv.d.QueryRW().CreateTrackStar(ctx, db.CreateTrackStarParams{
		TrackID:   trackUUID,
		UserID:    user.Uuid,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		logg.Error(ctx, "failed to create track star", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleUnstarTrack handles DELETE /tracks/{uuid}/star.
// Authenticated users may remove their own star from any track.
// No visibility check is performed; users can always remove stars from tracks
// that have since become private.
func (sv *server) handleUnstarTrack(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.MustGetUser(ctx)
	trackUUID := chi.URLParam(r, "uuid")

	n, err := sv.d.QueryRW().DeleteTrackStar(ctx, db.DeleteTrackStarParams{
		TrackID: trackUUID,
		UserID:  user.Uuid,
	})
	if err != nil {
		logg.Error(ctx, "failed to delete track star", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	if n == 0 {
		writeError(w, http.StatusNotFound, "star not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGetUserStars handles GET /users/{uuid}/stars.
// Returns all tracks starred by the given user that are visible to the caller.
// Anonymous callers see only stars on public tracks.
func (sv *server) handleGetUserStars(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	viewer := session.GetUser(ctx)
	userUUID := chi.URLParam(r, "uuid")

	var viewerID string
	if viewer != nil {
		viewerID = viewer.Uuid
	}

	tracks, err := sv.d.GetStarredTracks(ctx, userUUID, viewerID)
	if err != nil {
		logg.Error(ctx, "failed to get starred tracks", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	trackUUIDs := make([]string, len(tracks))
	for i, t := range tracks {
		trackUUIDs[i] = t.Uuid
	}

	tagsByTrack, err := sv.d.GetTagsForTracks(ctx, trackUUIDs)
	if err != nil {
		logg.Error(ctx, "failed to get tags for starred tracks", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	responses := make([]trackResponse, len(tracks))
	for i, t := range tracks {
		tags := tagsByTrack[t.Uuid]
		if tags == nil {
			tags = []string{}
		}
		responses[i] = trackResponseFromDB(t.Track, tags, t.Starred)
	}

	writeJSON(w, http.StatusOK, responses)
}
