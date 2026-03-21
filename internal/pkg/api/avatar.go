package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"codeberg.org/Codeberg/avatars"
	"github.com/go-chi/chi/v5"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/password"
	"jo-m.ch/go/detour/internal/pkg/session"
)

const avatarSeedLen = 16

// handleGetUserAvatar serves the SVG avatar for the user with the given UUID.
// This endpoint is public and requires no authentication.
func (sv *server) handleGetUserAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userUUID := chi.URLParam(r, "uuid")
	u, err := sv.d.QueryRO().GetUser(ctx, userUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		logg.Error(ctx, "failed to get user for avatar", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	eTag := fmt.Sprintf(`"%s-v0"`, u.AvatarSeed)
	if r.Header.Get("If-None-Match") == eTag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	svg := []byte(avatars.MakeAvatar(u.AvatarSeed))
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Header().Set("ETag", eTag)
	w.Header().Set("Content-Length", strconv.Itoa(len(svg)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(svg)
}

// handleRotateAvatar generates a new random avatar seed for the current user
// and returns the updated user response.
func (sv *server) handleRotateAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.GetUser(ctx)

	newSeed := password.GenRandAlnumString(avatarSeedLen)

	_, err := sv.d.QueryRW().UpdateUserAvatarSeed(ctx, db.UpdateUserAvatarSeedParams{
		UpdatedAt:  time.Now().UTC(),
		AvatarSeed: newSeed,
		Uuid:       user.Uuid,
	})
	if err != nil {
		logg.Error(ctx, "failed to rotate avatar seed", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	u, err := sv.d.QueryRO().GetUser(ctx, user.Uuid)
	if err != nil {
		logg.Error(ctx, "failed to get user after avatar rotation", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, userResponse{
		UUID:       u.Uuid,
		Email:      u.Email,
		Name:       u.Name,
		Admin:      u.Admin != 0,
		AvatarSeed: u.AvatarSeed,
	})
}
