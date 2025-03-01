package svc

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"goweb/internal/pkg/db"
	"goweb/internal/pkg/svc/tpl"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	netmiddleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/oapi-codegen/runtime/types"
)

//go:generate go tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config api-cfg.yaml api.yaml

//go:generate go tool qtc -dir=.

//go:embed api.yaml
var schema []byte
var Schema *openapi3.T

func init() {
	var err error
	Schema, err = openapi3.NewLoader().LoadFromData(schema)
	if err != nil {
		panic(err)
	}
}

type Server struct {
	db *sql.DB
}

func NewServer(db *sql.DB) Server {
	return Server{db: db}
}

// Compile time interface check.
var _ StrictServerInterface = (*Server)(nil)

const sessionCookieName = "session"

type SessionCookieWriter struct {
	value string
}

func (c *SessionCookieWriter) VisitPostApiV1LoginResponse(w http.ResponseWriter) error {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    c.value,
		Quoted:   true,
		MaxAge:   1800,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	fmt.Fprint(w, "Set cookie")

	return nil
}

/*
curl -X POST --data '{"username":"test","password":"asdf"}' -H 'content-type: application/json' http://127.0.0.1:8050/api/v1/login
*/
func (s *Server) PostApiV1Login(ctx context.Context, request PostApiV1LoginRequestObject) (PostApiV1LoginResponseObject, error) {
	// TODO: password check.
	fmt.Println(request.Body)
	return &SessionCookieWriter{value: "asdf"}, nil
}

func (s *Server) GetApiV1Users(ctx context.Context, request GetApiV1UsersRequestObject) (GetApiV1UsersResponseObject, error) {
	users, err := db.New(s.db).GetUsers(ctx)
	if err != nil {
		return GetApiV1Users500JSONResponse{}, nil
	}

	ret := GetApiV1Users200JSONResponse{}
	for _, u := range users {
		ret = append(ret, User{
			Username: u.Username,
			Email:    types.Email(u.Email),
		})
	}

	return ret, nil
}

var (
	HeaderContentType = http.CanonicalHeaderKey("Content-Type")
)

const (
	ApplicationJSON = "application/json"
	TextHTML        = "text/html"
)

func WantsJSON(acceptHeader string) bool {
	// TODO: make this more RFC compliant
	return strings.Contains(acceptHeader, ApplicationJSON)
}

func Body(w func(io.Writer)) io.Reader {
	reader, writer := io.Pipe()
	go func() {
		w(writer)
		writer.Close()
	}()

	return reader
}

/*
curl 'http://127.0.0.1:8050/api/v1/users/test' \
 -H 'Accept: application/json' \
 -H 'Cookie: session="asdf"'

curl 'http://127.0.0.1:8050/api/v1/users/test' \
 -H 'Cookie: session="asdf"'
*/

func (s *Server) GetApiV1UsersName(ctx context.Context, request GetApiV1UsersNameRequestObject) (GetApiV1UsersNameResponseObject, error) {
	user, err := db.New(s.db).GetUserByName(ctx, request.Name)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return GetApiV1UsersName404JSONResponse{}, nil
	}
	if err != nil {
		return GetApiV1UsersName500JSONResponse{}, nil
	}

	retUser := User{
		Username: user.Username,
		Email:    types.Email(user.Email),
	}
	if WantsJSON(request.Params.Accept) {
		ret := GetApiV1UsersName200JSONResponse(retUser)
		return ret, nil
	}

	p := tpl.MainPage{
		Username: user.Username,
	}

	return GetApiV1UsersName200TexthtmlResponse{Body: Body(func(w io.Writer) { tpl.WritePageTemplate(w, &p) })}, nil
}

func (s *Server) PostApiV1Users(ctx context.Context, request PostApiV1UsersRequestObject) (PostApiV1UsersResponseObject, error) {
	panic("unimplemented")
}

func (s *Server) PutApiV1UsersName(ctx context.Context, request PutApiV1UsersNameRequestObject) (PutApiV1UsersNameResponseObject, error) {
	panic("unimplemented")
}

func auth(ctx context.Context, a *openapi3filter.AuthenticationInput) error {
	if a.SecuritySchemeName != "CookieAuth" {
		panic("unknown security scheme")
	}

	cookie, err := a.RequestValidationInput.Request.Cookie(sessionCookieName)
	if err != nil {
		return errors.New("no session")
	}

	if cookie.Value != "asdf" {
		return errors.New("unknown session")
	}

	fmt.Println("authenticated")

	return nil
}

func New(db *sql.DB) http.Handler {
	mux := chi.NewRouter()
	mux.Use(middleware.RequestID)
	mux.Use(middleware.RealIP)
	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)
	mux.Use(middleware.Timeout(60 * time.Second))

	oapimux := chi.NewRouter()
	options := netmiddleware.Options{
		Options: openapi3filter.Options{
			AuthenticationFunc:    auth,
			ExcludeRequestBody:    false,
			ExcludeResponseBody:   false,
			IncludeResponseStatus: true,
			MultiError:            true,
		},
	}
	oapimux.Use(netmiddleware.OapiRequestValidatorWithOptions(Schema, &options))
	sv := NewServer(db)
	h := HandlerFromMux(NewStrictHandler(&sv, nil), oapimux)
	mux.Mount("/", h)

	return mux
}
