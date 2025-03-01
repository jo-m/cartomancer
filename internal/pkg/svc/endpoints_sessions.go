package svc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"goweb/internal/pkg/db"
	"goweb/internal/pkg/password"
	"goweb/internal/pkg/svc/tpl"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const sessionCookieName = "session"

type SessionCookieWriter struct {
	cookie http.Cookie
}

func (c *SessionCookieWriter) VisitPostApiV1SessionsLoginResponse(w http.ResponseWriter) error {
	http.SetCookie(w, &c.cookie)
	return nil
}

func (c *SessionCookieWriter) VisitPostApiV1SessionsLogoutResponse(w http.ResponseWriter) error {
	http.SetCookie(w, &c.cookie)
	return nil
}

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
		slog.Warn("Login failed", "email", user.Email)
		return PostApiV1SessionsLogin401JSONResponse{}, nil
	}
	slog.Info("Login succeeded", "email", user.Email)

	session := db.CreateUserSessionParams{
		Secret:    password.GenRandAlnumString(96),
		CreatedAt: time.Now(),
		UserID:    user.ID,
	}
	created, err := s.DB().CreateUserSession(ctx, session)
	if err != nil {
		slog.Warn("Creating session failed", "err", err)
		return PostApiV1SessionsLogin500JSONResponse{}, nil
	}

	cookie := http.Cookie{
		Name:     sessionCookieName,
		Value:    fmt.Sprintf("%d:%s", created.ID, created.Secret),
		MaxAge:   1800, // TODO: adjust
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	}
	return &SessionCookieWriter{cookie}, nil
}

/*
	curl -v -X POST http://127.0.0.1:8050/api/v1/sessions/logout \
		--cookie-jar cookies.txt
*/
func (s *Server) PostApiV1SessionsLogout(ctx context.Context, request PostApiV1SessionsLogoutRequestObject) (PostApiV1SessionsLogoutResponseObject, error) {
	cookie := http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Quoted:   true,
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	return &SessionCookieWriter{cookie}, nil
}
