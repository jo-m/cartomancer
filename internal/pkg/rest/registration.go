package rest

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/jobs"
	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/mail"
	"jo-m.ch/go/detour/internal/pkg/password"
	"jo-m.ch/go/detour/internal/pkg/session"
)

type registerRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type confirmTokenRequest struct {
	Token string `json:"token"`
}

type msgResponse struct {
	Msg string `json:"msg"`
}

var errEmailTaken = errors.New("email already taken")

func (sv *server) handleRegister(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req registerRequest
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
	if req.Password == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}

	hash, err := password.Hash(req.Password)
	if errors.Is(err, password.ErrTooLong) {
		writeError(w, http.StatusBadRequest, "password too long")
		return
	}
	if err != nil {
		logg.Error(ctx, "failed to hash password", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	userID, err := uuid.NewV7()
	if err != nil {
		logg.Error(ctx, "failed to generate uuid", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
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
		// Check if email is already taken.
		_, txErr := q.GetUserByEmail(ctx, req.Email)
		if txErr == nil {
			return errEmailTaken
		}
		if !errors.Is(txErr, sql.ErrNoRows) {
			return txErr
		}

		_, txErr = q.CreateUser(ctx, db.CreateUserParams{
			Uuid:           userID.String(),
			CreatedAt:      now,
			UpdatedAt:      now,
			Email:          req.Email,
			Name:           req.Name,
			PasswordHash:   hash,
			Admin:          0,
			EmailConfirmed: 0,
		})
		if txErr != nil {
			return txErr
		}

		_, txErr = q.CreateEmailVerification(ctx, db.CreateEmailVerificationParams{
			Uuid:      verID.String(),
			CreatedAt: now,
			ExpiresAt: now.Add(sv.appConfig.EmailVerificationExpiry),
			UserID:    userID.String(),
			Email:     req.Email,
		})
		return txErr
	})
	if errors.Is(err, errEmailTaken) {
		writeError(w, http.StatusConflict, "email already taken")
		return
	}
	if err != nil {
		logg.Error(ctx, "failed to register user", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	token, err := signEmailToken(verID.String(), sv.appConfig.EmailVerificationExpiry, sv.emailJWTSecret, sv.appConfig.AppName)
	if err != nil {
		logg.Error(ctx, "failed to sign email verification token", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	confirmURL := fmt.Sprintf("%s/confirm-email?token=%s", sv.appConfig.ExternalBaseURL, token)
	err = jobs.Submit(ctx, sv.jobSubmitter, mail.Args{
		To:      []string{req.Email},
		Subject: "Confirm your email",
		Body:    fmt.Sprintf("Please confirm your email by visiting:\n\n%s\n", confirmURL),
	}, jobs.Params{MaxRetries: 3})
	if err != nil {
		logg.Error(ctx, "failed to submit confirmation email job", "err", err)
		// Don't fail the request; user was created. They can request again.
	}

	writeJSON(w, http.StatusCreated, msgResponse{Msg: "check your email"})
}

var errNewEmailTaken = errors.New("new email already taken")

func (sv *server) handleConfirmEmail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req confirmTokenRequest
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if req.Token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	verUUID, err := verifyEmailToken(req.Token, sv.emailJWTSecret, sv.appConfig.AppName)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			writeError(w, http.StatusGone, "token expired")
			return
		}
		writeError(w, http.StatusNotFound, "invalid token")
		return
	}

	err = sv.d.WithTx(ctx, func(q *db.Queries) error {
		ver, txErr := q.GetEmailVerification(ctx, verUUID)
		if txErr != nil {
			return txErr
		}

		user, txErr := q.GetUser(ctx, ver.UserID)
		if txErr != nil {
			return txErr
		}

		if ver.Email == user.Email {
			// Registration confirmation: just mark as confirmed.
			_, txErr = q.ConfirmUserEmail(ctx, db.ConfirmUserEmailParams{
				UpdatedAt: time.Now().UTC(),
				Uuid:      ver.UserID,
			})
		} else {
			// Email change: check availability, then update.
			_, txErr = q.GetUserByEmail(ctx, ver.Email)
			if txErr == nil {
				return errNewEmailTaken
			}
			if !errors.Is(txErr, sql.ErrNoRows) {
				return txErr
			}
			_, txErr = q.UpdateUserEmail(ctx, db.UpdateUserEmailParams{
				Email:     ver.Email,
				UpdatedAt: time.Now().UTC(),
				Uuid:      ver.UserID,
			})
		}
		if txErr != nil {
			return txErr
		}

		_, txErr = q.DeleteEmailVerification(ctx, ver.Uuid)
		return txErr
	})
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "invalid token")
		return
	}
	if errors.Is(err, errNewEmailTaken) {
		writeError(w, http.StatusConflict, "email already taken")
		return
	}
	if err != nil {
		logg.Error(ctx, "failed to confirm email", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	// If the current session is authenticated, destroy it so the user must log in fresh.
	oldSess := session.MustGet(ctx)
	if oldSess.UserID.Valid {
		if delErr := session.Delete(ctx, &oldSess); delErr != nil {
			logg.Error(ctx, "failed to delete session after confirmation", "err", delErr)
		}
	}

	logg.Info(ctx, "email confirmed", "verification", verUUID)
	writeJSON(w, http.StatusOK, msgResponse{Msg: "email confirmed"})
}
