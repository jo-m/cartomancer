package endpoints

//lint:file-ignore ST1020,ST1003 Ignore generated method names.

import (
	"context"
	"database/sql"
	"errors"
	"goweb/internal/pkg/endpoints/tpl"
	"goweb/internal/pkg/logg"
	"goweb/internal/pkg/oapi"
	"goweb/internal/pkg/password"
	"goweb/internal/pkg/session"
)

/*
	curl -v http://127.0.0.1:8050/sessions/login \
		--cookie-jar cookies.txt --cookie cookies.txt
*/
func (s *Server) GetSessionsLogin(ctx context.Context, request oapi.GetSessionsLoginRequestObject) (oapi.GetSessionsLoginResponseObject, error) {
	p := tpl.LoginPage{
		BasePage: basePage(ctx),
	}
	return oapi.GetSessionsLogin200TexthtmlResponse{Body: renderPage(&p)}, nil
}

/*
	curl -v -X POST http://127.0.0.1:8050/sessions/login \
		--cookie-jar cookies.txt --cookie cookies.txt \
		-H "Content-Type: application/x-www-form-urlencoded" \
		-d "email=test@example.org&password=asdf"
*/
func (s *Server) PostSessionsLogin(ctx context.Context, request oapi.PostSessionsLoginRequestObject) (oapi.PostSessionsLoginResponseObject, error) {
	p := tpl.LoginPage{
		BasePage:  basePage(ctx),
		LoginData: *request.Body,
	}
	p.LoginData.Password = ""

	user, err := s.d.QueryRO().GetUserByEmail(ctx, request.Body.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			p.Message = "User not found"
			return oapi.PostSessionsLogin401TexthtmlResponse{Body: renderPage(&p)}, nil
		}

		logg.Warn(ctx, "Error fetching user", "email", user.Email, "err", err)
		return oapi.PostSessionsLogin500TexthtmlResponse{Body: renderError500(ctx)}, nil
	}

	// Check password.
	if !password.Check(request.Body.Password, user.PasswordHash) {
		logg.Warn(ctx, "Authentication failed", "email", user.Email)
		p.Message = "Authentication failed"
		return oapi.PostSessionsLogin401TexthtmlResponse{Body: renderPage(&p)}, nil
	}
	logg.Info(ctx, "Login succeeded", "user", user.ID)

	// Create new session.
	oldSess := session.MustGet(ctx)
	sess, err := session.Create(ctx, sql.NullString{Valid: true, String: user.ID}, &oldSess)
	if err != nil {
		logg.Warn(ctx, "Creating session failed", "err", err)
		return oapi.PostSessionsLogin500TexthtmlResponse{Body: renderError500(ctx)}, nil
	}
	logg.Debug(ctx, "Created new session", "id", sess.ID)

	// TODO: session message/flash
	// TODO: better redirection target
	return oapi.PostSessionsLogin302Response{
		Headers: oapi.PostSessionsLogin302ResponseHeaders{
			Location: "/",
		},
	}, nil
}

/*
	curl -v http://127.0.0.1:8050/sessions/logout \
		--cookie-jar cookies.txt --cookie cookies.txt
*/
func (s *Server) GetSessionsLogout(ctx context.Context, request oapi.GetSessionsLogoutRequestObject) (oapi.GetSessionsLogoutResponseObject, error) {
	p := tpl.LogoutPage{
		BasePage: basePage(ctx),
	}
	return oapi.GetSessionsLogout200TexthtmlResponse{Body: renderPage(&p)}, nil
}

/*
	curl -v -X POST http://127.0.0.1:8050/sessions/logout \
		--cookie-jar cookies.txt --cookie cookies.txt
*/
func (s *Server) PostSessionsLogout(ctx context.Context, request oapi.PostSessionsLogoutRequestObject) (oapi.PostSessionsLogoutResponseObject, error) {
	sess := session.MustGet(ctx)
	err := session.Delete(ctx, &sess)
	if err != nil {
		logg.Warn(ctx, "Logout failed", "err", err)
		return oapi.PostSessionsLogout500TexthtmlResponse{Body: renderError500(ctx)}, nil
	}
	logg.Info(ctx, "Logout succeeded", "session", sess.ID)

	// TODO: session message/flash
	// TODO: better redirection target
	return oapi.PostSessionsLogout302Response{
		Headers: oapi.PostSessionsLogout302ResponseHeaders{
			Location: "/",
		},
	}, nil
}
