package api

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"goweb/internal/pkg/db"
	"io"
	"strings"

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

// GetUsers implements StrictServerInterface.
func (s *Server) GetApiV1Users(ctx context.Context, request GetApiV1UsersRequestObject) (GetApiV1UsersResponseObject, error) {
	users, err := db.New(s.db).GetUsers(ctx)
	if err != nil {
		return GetApiV1Users500Response{}, nil
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

func WantsJSON(acceptHeader string) bool {
	// TODO: make this more RFC compliant
	return strings.Contains(acceptHeader, "application/json")
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

// GetUsersName implements StrictServerInterface.
func (s *Server) GetApiV1UsersName(ctx context.Context, request GetApiV1UsersNameRequestObject) (GetApiV1UsersNameResponseObject, error) {
	user, err := db.New(s.db).GetUserByName(ctx, request.Name)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return GetApiV1UsersName404Response{}, nil
	}
	if err != nil {
		return GetApiV1UsersName500Response{}, nil
	}

	retUser := User{
		Username: user.Username,
		Email:    types.Email(user.Email),
	}
	if WantsJSON(request.Params.Accept) {
		ret := GetApiV1UsersName200JSONResponse(retUser)
		return ret, nil
	}

	return GetApiV1UsersName200TexthtmlResponse{Body: Body(func(w io.Writer) { WriteHTMLUser(w, retUser) })}, nil
}

// PostUsers implements StrictServerInterface.
func (s *Server) PostApiV1Users(ctx context.Context, request PostApiV1UsersRequestObject) (PostApiV1UsersResponseObject, error) {
	panic("unimplemented")
}

// PutUsersName implements StrictServerInterface.
func (s *Server) PutApiV1UsersName(ctx context.Context, request PutApiV1UsersNameRequestObject) (PutApiV1UsersNameResponseObject, error) {
	panic("unimplemented")
}
