package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"jo-m.ch/go/cartomancer/internal/pkg/avatar"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/password"
	"jo-m.ch/go/cartomancer/internal/pkg/session"
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

	isAdmin := u.Admin != 0

	eTag := fmt.Sprintf(`"%s-%t-v1"`, u.AvatarSeed, isAdmin)
	if r.Header.Get(headerIfNoneMatch) == eTag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	svg := []byte(avatar.MakeAvatar(u.AvatarSeed, isAdmin))
	w.Header().Set(headerContentType, "image/svg+xml")
	w.Header().Set(headerCacheControl, "public, max-age=86400")
	w.Header().Set(headerETag, eTag)
	w.Header().Set(headerContentLength, strconv.Itoa(len(svg)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(svg)
}

// handleRotateAvatar generates a new random avatar seed for the current user
// and returns the updated user response.
func (sv *server) handleRotateAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.MustGetUser(ctx)

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

	writeJSON(w, http.StatusOK, makeUserResponse(&u))
}
