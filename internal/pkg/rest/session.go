package rest

import (
	"database/sql"
	"errors"
	"net/http"

	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/password"
	"jo-m.ch/go/detour/internal/pkg/session"
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
	UUID  string `json:"uuid"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Admin bool   `json:"admin"`
}

func (sv *server) handleLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if session.GetUser(ctx) != nil {
		writeError(w, http.StatusConflict, "already logged in")
		return
	}

	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := sv.d.QueryRO().GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		logg.Error(ctx, "failed to fetch user", "email", req.Email, "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if !password.Check(req.Password, user.PasswordHash) {
		logg.Warn(ctx, "invalid password", "email", user.Email)
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	logg.Info(ctx, "login succeeded", "user", user.Uuid)

	oldSess := session.MustGet(ctx)
	sess, err := session.Create(ctx, sql.NullString{Valid: true, String: user.Uuid}, &oldSess)
	if err != nil {
		logg.Error(ctx, "creating session failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	logg.Debug(ctx, "created new session", "id", sess.Uuid)

	writeJSON(w, http.StatusOK, sessionResponse{
		SessionUUID: sess.Uuid,
		User: &userResponse{
			UUID:  user.Uuid,
			Email: user.Email,
			Name:  user.Name,
			Admin: user.Admin != 0,
		},
	})
}

func (sv *server) handleLogout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sess := session.MustGet(ctx)
	err := session.Delete(ctx, &sess)
	if err != nil {
		logg.Error(ctx, "logout failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	logg.Info(ctx, "logout succeeded", "session", sess.Uuid)

	w.WriteHeader(http.StatusNoContent)
}

func (sv *server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sess := session.MustGet(ctx)
	resp := sessionResponse{
		SessionUUID: sess.Uuid,
	}

	if user := session.GetUser(ctx); user != nil {
		resp.User = &userResponse{
			UUID:  user.Uuid,
			Email: user.Email,
			Name:  user.Name,
			Admin: user.Admin != 0,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}
