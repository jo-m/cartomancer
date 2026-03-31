package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/session"
)

type trackGroupEntry struct {
	UUID        string `json:"uuid"`
	MemberCount int64  `json:"memberCount"`
	SampleName  string `json:"sampleName"`
}

type listTrackGroupsResponse struct {
	Groups []trackGroupEntry `json:"groups"`
}

type trackGroupDetailResponse struct {
	UUID   string          `json:"uuid"`
	Tracks []trackResponse `json:"tracks"`
}

// handleListTrackGroups returns all track groups for the authenticated user,
// excluding groups that have only one member.
func (sv *server) handleListTrackGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.GetUser(ctx)

	groups, err := sv.d.QueryRO().ListTrackGroupsWithCountByUser(ctx, user.Uuid)
	if err != nil {
		logg.Error(ctx, "failed to list track groups", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	entries := make([]trackGroupEntry, len(groups))
	for i, g := range groups {
		entries[i] = trackGroupEntry{
			UUID:        g.Uuid,
			MemberCount: g.MemberCount,
			SampleName:  g.SampleName,
		}
	}

	writeJSON(w, http.StatusOK, listTrackGroupsResponse{Groups: entries})
}

// handleGetTrackGroup returns the member tracks of a single track group.
func (sv *server) handleGetTrackGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.GetUser(ctx)
	groupUUID := chi.URLParam(r, "uuid")

	group, err := sv.d.QueryRO().GetTrackGroupByUUID(ctx, groupUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "group not found")
			return
		}
		logg.Error(ctx, "failed to get track group", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	if group.UserID != user.Uuid {
		writeError(w, http.StatusNotFound, "group not found")
		return
	}

	memberUUIDs, err := sv.d.QueryRO().ListTrackGroupMemberUUIDs(ctx, groupUUID)
	if err != nil {
		logg.Error(ctx, "failed to list group member uuids", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	tracks, err := sv.d.GetTracksByUUIDs(ctx, memberUUIDs, user.Uuid)
	if err != nil {
		logg.Error(ctx, "failed to get group member tracks", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	trackUUIDs := make([]string, len(tracks))
	for i, t := range tracks {
		trackUUIDs[i] = t.Uuid
	}
	tagsByTrack, err := sv.d.GetTagsForTracks(ctx, trackUUIDs)
	if err != nil {
		logg.Error(ctx, "failed to get tags for group tracks", "err", err)
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

	writeJSON(w, http.StatusOK, trackGroupDetailResponse{
		UUID:   groupUUID,
		Tracks: responses,
	})
}
