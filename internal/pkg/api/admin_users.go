package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/jobs"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/mail"
	"jo-m.ch/go/cartomancer/internal/pkg/password"
	"jo-m.ch/go/cartomancer/internal/pkg/session"
)

// Limits for the admin "send arbitrary email" endpoint.
const (
	adminEmailSubjectMaxLen = 256
	adminEmailBodyMaxLen    = 16 * 1024
)

type adminUserResponse struct {
	UUID                        string  `json:"uuid"`
	Email                       string  `json:"email"`
	Name                        string  `json:"name"`
	Admin                       bool    `json:"admin"`
	CreatedAt                   string  `json:"createdAt"`
	UpdatedAt                   string  `json:"updatedAt"`
	LastLoginAt                 *string `json:"lastLoginAt,omitempty"`
	LastActiveAt                *string `json:"lastActiveAt,omitempty"`
	HasPendingEmailVerification bool    `json:"hasPendingEmailVerification"`
	TrackCount                  int64   `json:"trackCount"`
}

func adminUserResponseFromDB(u db.User) adminUserResponse {
	resp := adminUserResponse{
		UUID:      u.Uuid,
		Email:     u.Email,
		Name:      u.Name,
		Admin:     u.Admin != 0,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
		UpdatedAt: u.UpdatedAt.Format(time.RFC3339),
	}
	if u.LastLoginAt.Valid {
		s := u.LastLoginAt.Time.Format(time.RFC3339)
		resp.LastLoginAt = &s
	}
	if u.LastActiveAt.Valid {
		s := u.LastActiveAt.Time.Format(time.RFC3339)
		resp.LastActiveAt = &s
	}
	return resp
}

// requireAdmin is middleware that allows only authenticated admin users.
func (sv *server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := session.GetUser(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		if user.Admin == 0 {
			writeError(w, http.StatusForbidden, "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (sv *server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ro := sv.d.QueryRO()

	users, err := ro.GetUsers(ctx)
	if err != nil {
		logg.Error(ctx, "failed to list users", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	pendingIDs, err := ro.GetUserIDsWithPendingEmailVerification(ctx)
	if err != nil {
		logg.Error(ctx, "failed to get pending email verifications", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}
	pendingSet := make(map[string]struct{}, len(pendingIDs))
	for _, id := range pendingIDs {
		pendingSet[id] = struct{}{}
	}

	resp := make([]adminUserResponse, len(users))
	for i, u := range users {
		resp[i] = adminUserResponseFromDB(db.User{
			Uuid:         u.Uuid,
			CreatedAt:    u.CreatedAt,
			UpdatedAt:    u.UpdatedAt,
			LastLoginAt:  u.LastLoginAt,
			LastActiveAt: u.LastActiveAt,
			Email:        u.Email,
			Name:         u.Name,
			Admin:        u.Admin,
		})
		_, resp[i].HasPendingEmailVerification = pendingSet[u.Uuid]
		resp[i].TrackCount = u.TrackCount
	}
	writeJSON(w, http.StatusOK, resp)
}

func (sv *server) handleAdminGetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ro := sv.d.QueryRO()

	userUUID := chi.URLParam(r, "uuid")
	u, err := ro.GetUser(ctx, userUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		logg.Error(ctx, "failed to get user", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	resp := adminUserResponseFromDB(u)
	_, err = ro.GetEmailVerificationByUserID(ctx, userUUID)
	if err == nil {
		resp.HasPendingEmailVerification = true
	} else if !errors.Is(err, sql.ErrNoRows) {
		logg.Error(ctx, "failed to check email verification", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

type adminCreateUserRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Admin bool   `json:"admin"`
}

type adminCreateUserResponse struct {
	adminUserResponse
	InitialPassword string `json:"initialPassword"`
}

func (sv *server) handleAdminCreateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req adminCreateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	req.Email = normalizeEmail(req.Email)
	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if !validateName(req.Name) {
		writeError(w, http.StatusBadRequest, "invalid name: 3-32 chars, letters/digits/underscores/hyphens only, no consecutive underscores or hyphens")
		return
	}

	id, err := uuid.NewV7()
	if err != nil {
		logg.Error(ctx, "failed to generate uuid", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	initialPassword := password.GenRandAlnumString(password.GeneratedPasswordLen)

	now := time.Now().UTC()
	var admin int64
	if req.Admin {
		admin = 1
	}

	hash, err := password.Hash(initialPassword)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid password: %s", err))
		return
	}

	var u db.User
	err = sv.d.WithTx(ctx, func(q *db.Queries) error {
		// Check email uniqueness.
		_, txErr := q.GetUserByEmail(ctx, req.Email)
		if txErr == nil {
			return errEmailTaken
		}
		if !errors.Is(txErr, sql.ErrNoRows) {
			return txErr
		}
		// Check name uniqueness (case-insensitive).
		_, txErr = q.GetUserByName(ctx, req.Name)
		if txErr == nil {
			return errNameTaken
		}
		if !errors.Is(txErr, sql.ErrNoRows) {
			return txErr
		}
		u, txErr = q.CreateUser(ctx, db.CreateUserParams{
			Uuid:           id.String(),
			CreatedAt:      now,
			UpdatedAt:      now,
			Email:          req.Email,
			Name:           req.Name,
			PasswordHash:   hash,
			Admin:          admin,
			EmailConfirmed: 1,
		})
		return txErr
	})
	if errors.Is(err, errEmailTaken) {
		writeError(w, http.StatusConflict, "email already taken")
		return
	}
	if errors.Is(err, errNameTaken) {
		writeError(w, http.StatusConflict, "name already taken")
		return
	}
	if err != nil {
		logg.Error(ctx, "failed to create user", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, adminCreateUserResponse{
		adminUserResponse: adminUserResponseFromDB(u),
		InitialPassword:   initialPassword,
	})
}

type adminUpdateUserRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Admin bool   `json:"admin"`
}

func (sv *server) handleAdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userUUID := chi.URLParam(r, "uuid")

	var req adminUpdateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	req.Email = normalizeEmail(req.Email)
	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if !validateName(req.Name) {
		writeError(w, http.StatusBadRequest, "invalid name: 3-32 chars, letters/digits/underscores/hyphens only, no consecutive underscores or hyphens")
		return
	}

	now := time.Now().UTC()
	var admin int64
	if req.Admin {
		admin = 1
	}

	// Admin guard check and update must be atomic to prevent TOCTOU races.
	err := sv.d.WithTx(ctx, func(q *db.Queries) error {
		existing, txErr := q.GetUser(ctx, userUUID)
		if txErr != nil {
			return txErr
		}
		// Prevent demoting the last admin.
		if existing.Admin != 0 && admin == 0 {
			adminCount, txErr := q.CountAdmins(ctx)
			if txErr != nil {
				return txErr
			}
			if adminCount <= 1 {
				return errLastAdmin
			}
		}
		// Check name uniqueness (case-insensitive, excluding this user).
		if !strings.EqualFold(existing.Name, req.Name) {
			_, txErr = q.GetUserByName(ctx, req.Name)
			if txErr == nil {
				return errNameTaken
			}
			if !errors.Is(txErr, sql.ErrNoRows) {
				return txErr
			}
		}
		// Check email uniqueness (excluding this user).
		// Both values are already lowercased via normalizeEmail.
		if existing.Email != req.Email {
			_, txErr = q.GetUserByEmail(ctx, req.Email)
			if txErr == nil {
				return errEmailTaken
			}
			if !errors.Is(txErr, sql.ErrNoRows) {
				return txErr
			}
		}
		_, txErr = q.UpdateUser(ctx, db.UpdateUserParams{
			UpdatedAt: now,
			Email:     req.Email,
			Name:      req.Name,
			Admin:     admin,
			Uuid:      userUUID,
		})
		if txErr != nil {
			return txErr
		}
		// Clean up any pending verifications when admin changes email.
		if existing.Email != req.Email {
			_, txErr = q.DeleteEmailVerificationsForUser(ctx, userUUID)
			if txErr != nil {
				return txErr
			}
		}
		return nil
	})
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if errors.Is(err, errLastAdmin) {
		writeError(w, http.StatusConflict, "cannot demote the last admin account")
		return
	}
	if errors.Is(err, errNameTaken) {
		writeError(w, http.StatusConflict, "name already taken")
		return
	}
	if errors.Is(err, errEmailTaken) {
		writeError(w, http.StatusConflict, "email already taken")
		return
	}
	if err != nil {
		logg.Error(ctx, "failed to update user", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	u, err := sv.d.QueryRO().GetUser(ctx, userUUID)
	if err != nil {
		logg.Error(ctx, "failed to get updated user", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, adminUserResponseFromDB(u))
}

func (sv *server) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userUUID := chi.URLParam(r, "uuid")

	currentUser := session.MustGetUser(ctx)
	if currentUser.Uuid == userUUID {
		writeError(w, http.StatusForbidden, "cannot delete your own account via admin endpoint")
		return
	}

	err := sv.d.WithTx(ctx, func(q *db.Queries) error {
		target, txErr := q.GetUser(ctx, userUUID)
		if txErr != nil {
			return txErr
		}
		if target.Admin != 0 {
			adminCount, txErr := q.CountAdmins(ctx)
			if txErr != nil {
				return txErr
			}
			if adminCount <= 1 {
				return errLastAdmin
			}
		}
		_, txErr = q.DeleteUser(ctx, userUUID)
		return txErr
	})
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
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

type adminResetPasswordRequest struct{}

type adminResetPasswordResponse struct {
	Password string `json:"password"`
}

func (sv *server) handleAdminResetUserPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userUUID := chi.URLParam(r, "uuid")

	currentUser := session.MustGetUser(ctx)
	if currentUser.Uuid == userUUID {
		writeError(w, http.StatusForbidden, "cannot reset your own password")
		return
	}

	var req adminResetPasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	newPassword := password.GenRandAlnumString(password.GeneratedPasswordLen)

	hash, err := password.Hash(newPassword)
	if err != nil {
		logg.Error(ctx, "failed to hash password", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	err = sv.d.WithTx(ctx, func(q *db.Queries) error {
		var txErr error
		_, txErr = q.GetUser(ctx, userUUID)
		if txErr != nil {
			return txErr
		}
		_, txErr = q.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
			UpdatedAt:    time.Now().UTC(),
			PasswordHash: hash,
			Uuid:         userUUID,
		})
		if txErr != nil {
			return txErr
		}
		_, txErr = q.DeleteAllUserSessions(ctx, sql.NullString{Valid: true, String: userUUID})
		return txErr
	})
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		logg.Error(ctx, "failed to reset password", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, adminResetPasswordResponse{Password: newPassword})
}

var errNoPendingVerification = errors.New("no pending email verification")
var errConfirmAdminForbidden = errors.New("cannot admin-confirm email for an admin user")

func (sv *server) handleAdminConfirmEmail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userUUID := chi.URLParam(r, "uuid")

	var updatedUser db.User

	err := sv.d.WithTx(ctx, func(q *db.Queries) error {
		target, txErr := q.GetUser(ctx, userUUID)
		if txErr != nil {
			return txErr
		}
		if target.Admin != 0 {
			return errConfirmAdminForbidden
		}
		ver, txErr := q.GetEmailVerificationByUserID(ctx, userUUID)
		if errors.Is(txErr, sql.ErrNoRows) {
			return errNoPendingVerification
		}
		if txErr != nil {
			return txErr
		}

		if ver.Email != target.Email {
			// Email change verification: check uniqueness before applying.
			_, txErr = q.GetUserByEmail(ctx, ver.Email)
			if txErr == nil {
				return errEmailTaken
			}
			if !errors.Is(txErr, sql.ErrNoRows) {
				return txErr
			}
			_, txErr = q.UpdateUserEmail(ctx, db.UpdateUserEmailParams{
				Email:     ver.Email,
				UpdatedAt: time.Now().UTC(),
				Uuid:      userUUID,
			})
		} else {
			// Registration confirmation: just mark as confirmed.
			_, txErr = q.ConfirmUserEmail(ctx, db.ConfirmUserEmailParams{
				UpdatedAt: time.Now().UTC(),
				Uuid:      userUUID,
			})
		}
		if txErr != nil {
			return txErr
		}

		_, txErr = q.DeleteEmailVerification(ctx, ver.Uuid)
		if txErr != nil {
			return txErr
		}

		updatedUser, txErr = q.GetUser(ctx, userUUID)
		return txErr
	})
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if errors.Is(err, errConfirmAdminForbidden) {
		writeError(w, http.StatusForbidden, "cannot admin-confirm email for an admin user")
		return
	}
	if errors.Is(err, errNoPendingVerification) {
		writeError(w, http.StatusNotFound, "no pending email verification")
		return
	}
	if errors.Is(err, errEmailTaken) {
		writeError(w, http.StatusConflict, "email already taken")
		return
	}
	if err != nil {
		logg.Error(ctx, "failed to confirm email", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, adminUserResponseFromDB(updatedUser))
}

// adminSendEmailRequest is the body for [server.handleAdminSendEmail].
type adminSendEmailRequest struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// containsLineBreak reports whether s contains a CR or LF character.
// Used to reject header injection attempts in the email subject.
func containsLineBreak(s string) bool {
	return strings.ContainsAny(s, "\r\n")
}

// handleAdminSendEmail sends an arbitrary plain-text email to a single registered user.
// The email is dispatched via the [mail.Mailer] job queue; this endpoint returns 202
// once the job has been submitted, not after delivery.
func (sv *server) handleAdminSendEmail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userUUID := chi.URLParam(r, "uuid")

	var req adminSendEmailRequest
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	req.Subject = strings.TrimSpace(req.Subject)
	if req.Subject == "" {
		writeError(w, http.StatusBadRequest, "subject is required")
		return
	}
	if len(req.Subject) > adminEmailSubjectMaxLen {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("subject must be at most %d characters", adminEmailSubjectMaxLen))
		return
	}
	if containsLineBreak(req.Subject) {
		writeError(w, http.StatusBadRequest, "subject must not contain line breaks")
		return
	}
	if req.Body == "" {
		writeError(w, http.StatusBadRequest, "body is required")
		return
	}
	if len(req.Body) > adminEmailBodyMaxLen {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("body must be at most %d characters", adminEmailBodyMaxLen))
		return
	}

	u, err := sv.d.QueryRO().GetUser(ctx, userUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		logg.Error(ctx, "failed to get user", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	err = jobs.Submit(ctx, sv.jobSubmitter, mail.Args{
		To:      []string{u.Email},
		Subject: req.Subject,
		Body:    req.Body,
	}, jobs.Params{MaxRetries: 5, BackofFactorS: 60 * time.Second})
	if err != nil {
		logg.Error(ctx, "failed to submit admin email job", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusAccepted, msgResponse{Msg: "email queued for delivery"})
}
