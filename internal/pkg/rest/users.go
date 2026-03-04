package rest

import (
	"errors"
	"net/http"
	"time"

	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/password"
	"jo-m.ch/go/detour/internal/pkg/session"
)

var errLastAdmin = errors.New("cannot delete the last admin account")

type updateMeRequest struct {
	Name string `json:"name"`
}

func (sv *server) handleUpdateAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.GetUser(ctx)

	var req updateMeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	_, err := sv.d.QueryRW().UpdateUserName(ctx, db.UpdateUserNameParams{
		UpdatedAt: time.Now().UTC(),
		Name:      req.Name,
		Uuid:      user.Uuid,
	})
	if err != nil {
		logg.Error(ctx, "failed to update user name", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	u, err := sv.d.QueryRO().GetUser(ctx, user.Uuid)
	if err != nil {
		logg.Error(ctx, "failed to get updated user", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, userResponse{
		UUID:  u.Uuid,
		Email: u.Email,
		Name:  u.Name,
		Admin: u.Admin != 0,
	})
}

func (sv *server) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.GetUser(ctx)
	sess := session.MustGet(ctx)

	// Admin check must happen before session deletion (which is irreversible).
	// The actual deletion tx below re-checks atomically to prevent TOCTOU races.
	if user.Admin != 0 {
		adminCount, err := sv.d.QueryRO().CountAdmins(ctx)
		if err != nil {
			logg.Error(ctx, "failed to count admins", "err", err)
			writeStatusError(w, http.StatusInternalServerError)
			return
		}
		if adminCount <= 1 {
			writeError(w, http.StatusConflict, "cannot delete the last admin account")
			return
		}
	}

	// Delete session (uses its own tx; cannot be nested inside the user-deletion tx).
	if err := session.Delete(ctx, &sess); err != nil {
		logg.Error(ctx, "failed to delete session", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	// Admin count check and user deletion must be atomic to prevent a race where two
	// admins both pass the guard and both delete themselves, leaving no admins.
	err := sv.d.WithTx(ctx, func(q *db.Queries) error {
		if user.Admin != 0 {
			adminCount, txErr := q.CountAdmins(ctx)
			if txErr != nil {
				return txErr
			}
			if adminCount <= 1 {
				return errLastAdmin
			}
		}
		_, txErr := q.DeleteUser(ctx, user.Uuid)
		return txErr
	})
	if errors.Is(err, errLastAdmin) {
		writeError(w, http.StatusConflict, "cannot delete the last admin account")
		return
	}
	if err != nil {
		logg.Error(ctx, "failed to delete user", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type changePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

func (sv *server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.GetUser(ctx)

	var req changePasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if req.OldPassword == "" {
		writeError(w, http.StatusBadRequest, "oldPassword is required")
		return
	}
	if req.NewPassword == "" {
		writeError(w, http.StatusBadRequest, "newPassword is required")
		return
	}

	u, err := sv.d.QueryRO().GetUser(ctx, user.Uuid)
	if err != nil {
		logg.Error(ctx, "failed to get user", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	if !password.Check(req.OldPassword, u.PasswordHash) {
		logg.Warn(ctx, "invalid old password for change-password", "user", user.Uuid)
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	_, err = sv.d.QueryRW().UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		UpdatedAt:    time.Now().UTC(),
		PasswordHash: password.Hash(req.NewPassword),
		Uuid:         user.Uuid,
	})
	if err != nil {
		logg.Error(ctx, "failed to update password", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
