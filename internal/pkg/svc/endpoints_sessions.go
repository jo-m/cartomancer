package svc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"goweb/internal/pkg/db"
	"goweb/internal/pkg/logging"
	"goweb/internal/pkg/password"
	"goweb/internal/pkg/session"
	"goweb/internal/pkg/svc/tpl"
	"io"
)

/*
	curl -v http://127.0.0.1:8050/api/v1/sessions/login \
		--cookie-jar cookies.txt --cookie cookies.txt
*/
func (s *Server) GetApiV1SessionsLogin(ctx context.Context, request GetApiV1SessionsLoginRequestObject) (GetApiV1SessionsLoginResponseObject, error) {
	p := tpl.LoginPage{}
	return GetApiV1SessionsLogin200TexthtmlResponse{Body: Body(func(w io.Writer) { tpl.WritePageTemplate(w, &p) })}, nil
}

/*
7
*/
func (s *Server) PostApiV1SessionsLogin(ctx context.Context, request PostApiV1SessionsLoginRequestObject) (PostApiV1SessionsLoginResponseObject, error) {
	user, err := s.q().GetUserByEmail(ctx, request.Body.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PostApiV1SessionsLogin401JSONResponse{}, err
		}

		return PostApiV1SessionsLogin500JSONResponse{}, err
	}

	if !password.Check(request.Body.Password, user.PasswordHash) {
		logging.Warn(ctx, "Authentication failed", "email", user.Email)
		return PostApiV1SessionsLogin401JSONResponse{}, nil
	}

	sess := session.MustGetSession(ctx)
	logging.Info(ctx, "Login succeeded", "user", user.ID, "session", sess.ID)
	err = s.q().SetSessionUserID(ctx, db.SetSessionUserIDParams{UserID: sql.NullString{Valid: true, String: user.ID}, ID: sess.ID})
	if err != nil {
		logging.Warn(ctx, "Session login failed", "err", err)
		return PostApiV1SessionsLogin500JSONResponse{}, nil
	}

	return PostApiV1SessionsLogin204Response{}, nil
}

/*
	curl -v http://127.0.0.1:8050/api/v1/sessions/logout \
		--cookie-jar cookies.txt --cookie cookies.txt
*/
func (s *Server) GetApiV1SessionsLogout(ctx context.Context, request GetApiV1SessionsLogoutRequestObject) (GetApiV1SessionsLogoutResponseObject, error) {
	p := tpl.LogoutPage{BasePage: tpl.BasePage{CurrentUserName: fmt.Sprint(session.MustGetUser(ctx).Email)}}
	return GetApiV1SessionsLogout200TexthtmlResponse{Body: Body(func(w io.Writer) { tpl.WritePageTemplate(w, &p) })}, nil
}

/*
	curl -v -X POST http://127.0.0.1:8050/api/v1/sessions/logout \
		--cookie-jar cookies.txt --cookie cookies.txt
*/
func (s *Server) PostApiV1SessionsLogout(ctx context.Context, request PostApiV1SessionsLogoutRequestObject) (PostApiV1SessionsLogoutResponseObject, error) {
	sess := session.MustGetSession(ctx)
	err := s.q().SetSessionUserID(ctx, db.SetSessionUserIDParams{ID: sess.ID})
	if err != nil {
		logging.Warn(ctx, "Session logout failed", "err", err)
		return PostApiV1SessionsLogout500JSONResponse{}, nil
	}
	logging.Info(ctx, "Logout succeeded", "session", sess.ID)

	return PostApiV1SessionsLogout204Response{}, nil
}
