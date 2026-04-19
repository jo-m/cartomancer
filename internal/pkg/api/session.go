package api

import (
	"database/sql"
	"errors"
	"net/http"

	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/password"
	"jo-m.ch/go/cartomancer/internal/pkg/session"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type sessionResponse struct {
	SessionUUID string        `json:"sessionUuid"`
	User        *userResponse `json:"user,omitempty"`
}

type userResponse struct {
	UUID         string   `json:"uuid"`
	Email        string   `json:"email"`
	Name         string   `json:"name"`
	Admin        bool     `json:"admin"`
	AvatarSeed   string   `json:"avatarSeed"`
	LocationName *string  `json:"locationName"`
	LocationLat  *float64 `json:"locationLat"`
	LocationLon  *float64 `json:"locationLon"`
}

// makeUserResponse builds a userResponse from a database user.
func makeUserResponse(u *db.User) userResponse {
	resp := userResponse{
		UUID:       u.Uuid,
		Email:      u.Email,
		Name:       u.Name,
		Admin:      u.Admin != 0,
		AvatarSeed: u.AvatarSeed,
	}
	if u.LocationName.Valid {
		resp.LocationName = &u.LocationName.String
		resp.LocationLat = &u.LocationLat.Float64
		resp.LocationLon = &u.LocationLon.Float64
	}
	return resp
}

func (sv *server) handleLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if session.GetUser(ctx) != nil {
		writeError(w, http.StatusConflict, "already logged in")
		return
	}

	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	req.Email = normalizeEmail(req.Email)

	user, err := sv.d.QueryRO().GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		logg.Error(ctx, "failed to fetch user", "email", req.Email, "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	if !password.Check(req.Password, user.PasswordHash) {
		logg.Warn(ctx, "invalid password", "email", user.Email)
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if user.EmailConfirmed == 0 {
		writeError(w, http.StatusForbidden, "email not confirmed")
		return
	}
	logg.Info(ctx, "login succeeded", "user", user.Uuid)

	oldSess := session.Get(ctx)
	sess, err := session.Create(ctx, sql.NullString{Valid: true, String: user.Uuid}, oldSess)
	if err != nil {
		logg.Error(ctx, "creating session failed", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}
	logg.Debug(ctx, "created new session", "id", sess.Uuid)

	ur := makeUserResponse(&user)
	writeJSON(w, http.StatusOK, sessionResponse{
		SessionUUID: sess.Uuid,
		User:        &ur,
	})
}

func (sv *server) handleLogout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sess := session.Get(ctx)
	if sess == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	err := session.Delete(ctx, sess)
	if err != nil {
		logg.Error(ctx, "logout failed", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}
	logg.Info(ctx, "logout succeeded", "session", sess.Uuid)

	w.WriteHeader(http.StatusNoContent)
}

func (sv *server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var resp sessionResponse
	if sess := session.Get(ctx); sess != nil {
		resp.SessionUUID = sess.Uuid
	}

	if user := session.GetUser(ctx); user != nil {
		ur := makeUserResponse(user)
		resp.User = &ur
	}

	writeJSON(w, http.StatusOK, resp)
}
