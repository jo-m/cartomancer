package svc

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"goweb/internal/pkg/db"
	"goweb/internal/pkg/svc/tpl"
	"io"

	"github.com/oapi-codegen/runtime/types"
)

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
