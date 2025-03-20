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
	curl -v http://127.0.0.1:8050/api/v1/sessions/login \
		--cookie-jar cookies.txt --cookie cookies.txt
*/
func (s *Server) GetApiV1SessionsLogin(ctx context.Context, request oapi.GetApiV1SessionsLoginRequestObject) (oapi.GetApiV1SessionsLoginResponseObject, error) {
	p := tpl.LoginPage{
		BasePage: BasePage(ctx),
	}
	return oapi.GetApiV1SessionsLogin200TexthtmlResponse{Body: RenderPage(&p)}, nil
}

/*
	curl -v -X POST http://127.0.0.1:8050/api/v1/sessions/login \
		--cookie-jar cookies.txt --cookie cookies.txt \
		-H "Content-Type: application/x-www-form-urlencoded" \
		-d "email=test@example.org&password=asdf"
*/
func (s *Server) PostApiV1SessionsLogin(ctx context.Context, request oapi.PostApiV1SessionsLoginRequestObject) (oapi.PostApiV1SessionsLoginResponseObject, error) {
	p := tpl.LoginPage{
		BasePage:  BasePage(ctx),
		LoginData: *request.Body,
	}
	p.LoginData.Password = ""

	user, err := s.d.QueryRO().GetUserByEmail(ctx, request.Body.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			p.Message = "User not found"
			return oapi.PostApiV1SessionsLogin401TexthtmlResponse{Body: RenderPage(&p)}, nil
		}

		logg.Warn(ctx, "Error fetching user", "email", user.Email, "err", err)
		return oapi.PostApiV1SessionsLogin500TexthtmlResponse{Body: RenderError500(ctx)}, nil
	}

	// Check password.
	if !password.Check(request.Body.Password, user.PasswordHash) {
		logg.Warn(ctx, "Authentication failed", "email", user.Email)
		p.Message = "Authentication failed"
		return oapi.PostApiV1SessionsLogin401TexthtmlResponse{Body: RenderPage(&p)}, nil
	}
	logg.Info(ctx, "Login succeeded", "user", user.ID)

	// Create new session.
	oldSess := session.MustGet(ctx)
	sess, err := session.Create(ctx, sql.NullString{Valid: true, String: user.ID}, &oldSess)
	if err != nil {
		logg.Warn(ctx, "Creating session failed", "err", err)
		return oapi.PostApiV1SessionsLogin500TexthtmlResponse{Body: RenderError500(ctx)}, nil
	}
	logg.Debug(ctx, "Created new session", "id", sess.ID)

	// TODO: session message/flash
	// TODO: better redirection target
	return oapi.PostApiV1SessionsLogin302Response{
		Headers: oapi.PostApiV1SessionsLogin302ResponseHeaders{
			Location: "/",
		},
	}, nil
}

/*
	curl -v http://127.0.0.1:8050/api/v1/sessions/logout \
		--cookie-jar cookies.txt --cookie cookies.txt
*/
func (s *Server) GetApiV1SessionsLogout(ctx context.Context, request oapi.GetApiV1SessionsLogoutRequestObject) (oapi.GetApiV1SessionsLogoutResponseObject, error) {
	p := tpl.LogoutPage{
		BasePage: BasePage(ctx),
	}
	return oapi.GetApiV1SessionsLogout200TexthtmlResponse{Body: RenderPage(&p)}, nil
}

/*
	curl -v -X POST http://127.0.0.1:8050/api/v1/sessions/logout \
		--cookie-jar cookies.txt --cookie cookies.txt
*/
func (s *Server) PostApiV1SessionsLogout(ctx context.Context, request oapi.PostApiV1SessionsLogoutRequestObject) (oapi.PostApiV1SessionsLogoutResponseObject, error) {
	sess := session.MustGet(ctx)
	err := session.Delete(ctx, &sess)
	if err != nil {
		logg.Warn(ctx, "Logout failed", "err", err)
		return oapi.PostApiV1SessionsLogout500TexthtmlResponse{Body: RenderError500(ctx)}, nil
	}
	logg.Info(ctx, "Logout succeeded", "session", sess.ID)

	// TODO: session message/flash
	// TODO: better redirection target
	return oapi.PostApiV1SessionsLogout302Response{
		Headers: oapi.PostApiV1SessionsLogout302ResponseHeaders{
			Location: "/",
		},
	}, nil
}
