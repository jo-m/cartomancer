package rest

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/jobs"
	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/mail"
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
		writeError(w, http.StatusForbidden, "incorrect password")
		return
	}

	hash, err := password.Hash(req.NewPassword)
	if errors.Is(err, password.ErrTooLong) {
		writeError(w, http.StatusBadRequest, "password too long")
		return
	}
	if err != nil {
		logg.Error(ctx, "failed to hash password", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	currentSession := session.MustGet(ctx)

	err = sv.d.WithTx(ctx, func(q *db.Queries) error {
		if _, txErr := q.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
			UpdatedAt:    time.Now().UTC(),
			PasswordHash: hash,
			Uuid:         user.Uuid,
		}); txErr != nil {
			return txErr
		}
		_, txErr := q.DeleteOtherUserSessions(ctx, db.DeleteOtherUserSessionsParams{
			UserID: currentSession.UserID,
			Uuid:   currentSession.Uuid,
		})
		return txErr
	})
	if err != nil {
		logg.Error(ctx, "failed to update password", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type changeEmailRequest struct {
	NewEmail string `json:"newEmail"`
	Password string `json:"password"`
}

func (sv *server) handleChangeEmail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := session.GetUser(ctx)

	var req changeEmailRequest
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	req.NewEmail = normalizeEmail(req.NewEmail)
	if req.NewEmail == "" {
		writeError(w, http.StatusBadRequest, "newEmail is required")
		return
	}
	if req.Password == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}

	u, err := sv.d.QueryRO().GetUser(ctx, user.Uuid)
	if err != nil {
		logg.Error(ctx, "failed to get user", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	if !password.Check(req.Password, u.PasswordHash) {
		writeError(w, http.StatusForbidden, "incorrect password")
		return
	}

	verID, err := uuid.NewV7()
	if err != nil {
		logg.Error(ctx, "failed to generate uuid", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	now := time.Now().UTC()

	err = sv.d.WithTx(ctx, func(q *db.Queries) error {
		// Check if new email is taken.
		_, txErr := q.GetUserByEmail(ctx, req.NewEmail)
		if txErr == nil {
			return errNewEmailTaken
		}
		if !errors.Is(txErr, sql.ErrNoRows) {
			return txErr
		}

		// Delete any existing verifications for this user.
		_, txErr = q.DeleteEmailVerificationsForUser(ctx, user.Uuid)
		if txErr != nil {
			return txErr
		}

		_, txErr = q.CreateEmailVerification(ctx, db.CreateEmailVerificationParams{
			Uuid:      verID.String(),
			CreatedAt: now,
			ExpiresAt: now.Add(verificationExpiry),
			UserID:    user.Uuid,
			Email:     req.NewEmail,
		})
		return txErr
	})
	if errors.Is(err, errNewEmailTaken) {
		writeError(w, http.StatusConflict, "email already taken")
		return
	}
	if err != nil {
		logg.Error(ctx, "failed to create email change verification", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	token, err := signEmailToken(verID.String(), verificationExpiry, sv.emailJWTSecret, sv.appConfig.AppName)
	if err != nil {
		logg.Error(ctx, "failed to sign email verification token", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	confirmURL := fmt.Sprintf("%s/confirm-email?token=%s", sv.appConfig.ExternalBaseURL, token)
	err = jobs.Submit(ctx, sv.jobSubmitter, mail.Args{
		To:      []string{req.NewEmail},
		Subject: "Confirm your new email",
		Body:    fmt.Sprintf("Please confirm your new email by visiting:\n\n%s\n", confirmURL),
	}, jobs.Params{MaxRetries: 3})
	if err != nil {
		logg.Error(ctx, "failed to submit email change confirmation job", "err", err)
	}

	writeJSON(w, http.StatusOK, msgResponse{Msg: "check your email"})
}
