package rest

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/jobs"
	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/mail"
	"jo-m.ch/go/detour/internal/pkg/password"
	"jo-m.ch/go/detour/internal/pkg/session"
)

const generatedPasswordLen = 20

type adminUserResponse struct {
	UUID         string  `json:"uuid"`
	Email        string  `json:"email"`
	Name         string  `json:"name"`
	Admin        bool    `json:"admin"`
	CreatedAt    string  `json:"createdAt"`
	UpdatedAt    string  `json:"updatedAt"`
	LastLoginAt  *string `json:"lastLoginAt,omitempty"`
	LastActiveAt *string `json:"lastActiveAt,omitempty"`
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

	users, err := sv.d.QueryRO().GetUsers(ctx)
	if err != nil {
		logg.Error(ctx, "failed to list users", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	resp := make([]adminUserResponse, len(users))
	for i, u := range users {
		resp[i] = adminUserResponseFromDB(u)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (sv *server) handleAdminGetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userUUID := chi.URLParam(r, "uuid")
	u, err := sv.d.QueryRO().GetUser(ctx, userUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		logg.Error(ctx, "failed to get user", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, adminUserResponseFromDB(u))
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
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	id, err := uuid.NewV7()
	if err != nil {
		logg.Error(ctx, "failed to generate uuid", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	initialPassword := password.GenRandPrintableString(generatedPasswordLen)

	now := time.Now().UTC()
	var admin int64
	if req.Admin {
		admin = 1
	}

	u, err := sv.d.QueryRW().CreateUser(ctx, db.CreateUserParams{
		Uuid:         id.String(),
		CreatedAt:    now,
		UpdatedAt:    now,
		Email:        req.Email,
		Name:         req.Name,
		PasswordHash: password.Hash(initialPassword),
		Admin:        admin,
	})
	if err != nil {
		logg.Error(ctx, "failed to create user", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
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
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	existing, err := sv.d.QueryRO().GetUser(ctx, userUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		logg.Error(ctx, "failed to get user", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if existing.Admin != 0 {
		writeError(w, http.StatusForbidden, "cannot modify admin accounts")
		return
	}

	now := time.Now().UTC()
	var admin int64
	if req.Admin {
		admin = 1
	}

	_, err = sv.d.QueryRW().UpdateUser(ctx, db.UpdateUserParams{
		UpdatedAt: now,
		Email:     req.Email,
		Name:      req.Name,
		Admin:     admin,
		Uuid:      userUUID,
	})
	if err != nil {
		logg.Error(ctx, "failed to update user", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	u, err := sv.d.QueryRO().GetUser(ctx, userUUID)
	if err != nil {
		logg.Error(ctx, "failed to get updated user", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, adminUserResponseFromDB(u))
}

func (sv *server) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userUUID := chi.URLParam(r, "uuid")

	target, err := sv.d.QueryRO().GetUser(ctx, userUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		logg.Error(ctx, "failed to get user", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if target.Admin != 0 {
		writeError(w, http.StatusForbidden, "cannot modify admin accounts")
		return
	}

	_, err = sv.d.QueryRW().DeleteUser(ctx, userUUID)
	if err != nil {
		logg.Error(ctx, "failed to delete user", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type adminResetPasswordRequest struct {
	SendEmail bool `json:"sendEmail"`
}

type adminResetPasswordResponse struct {
	Password string `json:"password"`
}

func (sv *server) handleAdminResetUserPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userUUID := chi.URLParam(r, "uuid")

	var req adminResetPasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	u, err := sv.d.QueryRO().GetUser(ctx, userUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		logg.Error(ctx, "failed to get user", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if u.Admin != 0 {
		writeError(w, http.StatusForbidden, "cannot modify admin accounts")
		return
	}

	newPassword := password.GenRandPrintableString(generatedPasswordLen)

	_, err = sv.d.QueryRW().UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		UpdatedAt:    time.Now().UTC(),
		PasswordHash: password.Hash(newPassword),
		Uuid:         userUUID,
	})
	if err != nil {
		logg.Error(ctx, "failed to reset password", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if req.SendEmail {
		err = jobs.Submit(ctx, sv.jobSubmitter, mail.Args{
			To:      []string{u.Email},
			Subject: "Your password has been reset",
			Body:    fmt.Sprintf("Hello %s,\n\nAn administrator has reset your password.\n\nYour new password is: %s\n\nPlease change it after logging in.\n", u.Name, newPassword),
		}, jobs.Params{MaxRetries: 3})
		if err != nil {
			logg.Error(ctx, "failed to submit password reset email job", "err", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	writeJSON(w, http.StatusOK, adminResetPasswordResponse{Password: newPassword})
}
