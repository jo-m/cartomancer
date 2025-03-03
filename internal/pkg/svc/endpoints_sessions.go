package svc

import (
	"context"
	"database/sql"
	"errors"
	"goweb/internal/pkg/db"
	"goweb/internal/pkg/password"
	"goweb/internal/pkg/session"
	"goweb/internal/pkg/svc/tpl"
	"io"
	"log/slog"
)

/*
http://127.0.0.1:8050/api/v1/sessions/login
*/
func (s *Server) GetApiV1SessionsLogin(ctx context.Context, request GetApiV1SessionsLoginRequestObject) (GetApiV1SessionsLoginResponseObject, error) {
	p := tpl.LoginPage{}
	return GetApiV1SessionsLogin200TexthtmlResponse{Body: Body(func(w io.Writer) { tpl.WritePageTemplate(w, &p) })}, nil
}

/*
	curl -v -X POST http://127.0.0.1:8050/api/v1/sessions/login \
		--cookie-jar cookies.txt \
		-H "Content-Type: application/x-www-form-urlencoded" \
		-d "email=test@example.org&password=asdf"
*/
func (s *Server) PostApiV1SessionsLogin(ctx context.Context, request PostApiV1SessionsLoginRequestObject) (PostApiV1SessionsLoginResponseObject, error) {
	user, err := s.DB().GetUserByEmail(ctx, request.Body.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PostApiV1SessionsLogin401JSONResponse{}, err
		}

		return PostApiV1SessionsLogin500JSONResponse{}, err
	}

	if !password.Check(request.Body.Password, user.PasswordHash) {
		slog.Warn("Authentication failed", "email", user.Email)
		return PostApiV1SessionsLogin401JSONResponse{}, nil
	}

	sessionID := session.GetSessionID(ctx)
	slog.Info("Authentication succeeded", "email", user.Email, "session", sessionID)
	err = s.DB().SetSessionUserID(ctx, db.SetSessionUserIDParams{UserID: sql.NullInt64{Valid: true, Int64: user.ID}, ID: sessionID})
	if err != nil {
		slog.Warn("Session login failed", "err", err)
		return PostApiV1SessionsLogin500JSONResponse{}, nil
	}

	return PostApiV1SessionsLogin204Response{}, nil
}

func (s *Server) GetApiV1SessionsLogout(ctx context.Context, request GetApiV1SessionsLogoutRequestObject) (GetApiV1SessionsLogoutResponseObject, error) {
	p := tpl.LogoutPage{}
	return GetApiV1SessionsLogout200TexthtmlResponse{Body: Body(func(w io.Writer) { tpl.WritePageTemplate(w, &p) })}, nil
}

/*
	curl -v -X POST http://127.0.0.1:8050/api/v1/sessions/logout \
		--cookie-jar cookies.txt
*/
func (s *Server) PostApiV1SessionsLogout(ctx context.Context, request PostApiV1SessionsLogoutRequestObject) (PostApiV1SessionsLogoutResponseObject, error) {
	sessionID := session.GetSessionID(ctx)
	err := s.DB().SetSessionUserID(ctx, db.SetSessionUserIDParams{ID: sessionID})
	if err != nil {
		slog.Warn("Session logout failed", "err", err)
		return PostApiV1SessionsLogout500JSONResponse{}, nil
	}
	slog.Info("Logout succeeded", "session", sessionID)

	return PostApiV1SessionsLogout204Response{}, nil
}
