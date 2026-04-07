package api

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/session"
)

var errTrackNotVisible = errors.New("track not visible")

// handleStarTrack handles POST /tracks/{uuid}/star.
// Authenticated users may star any track visible to them.
// Starring a track that is already starred is idempotent.
func (sv *server) handleStarTrack(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.MustGetUser(ctx)
	trackUUID := chi.URLParam(r, "uuid")

	err := sv.d.WithTx(ctx, func(q *db.Queries) error {
		t, txErr := q.GetTrackByUUID(ctx, trackUUID)
		if txErr != nil {
			return txErr
		}
		if t.Public == 0 && t.UserID != user.Uuid {
			return errTrackNotVisible
		}
		return q.CreateTrackStar(ctx, db.CreateTrackStarParams{
			TrackID:   trackUUID,
			UserID:    user.Uuid,
			CreatedAt: time.Now().UTC(),
		})
	})
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, errTrackNotVisible) {
		writeError(w, http.StatusNotFound, "track not found")
		return
	}
	if err != nil {
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
		responses[i] = trackResponseFromDB(t, tags, nil, viewer != nil && viewer.Uuid == t.UserID)
	}

	writeJSON(w, http.StatusOK, responses)
}
