package svc

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"goweb/internal/pkg/db"
	"goweb/internal/pkg/session"
	"goweb/internal/pkg/svc/tpl"
	"io"

	"github.com/oapi-codegen/runtime/types"
)

/*
	curl -v 'http://127.0.0.1:8050/api/v1/users' \
		--cookie-jar cookies.txt --cookie cookies.txt
*/
func (s *Server) GetApiV1Users(ctx context.Context, request GetApiV1UsersRequestObject) (GetApiV1UsersResponseObject, error) {
	users, err := db.New(s.db).GetUsers(ctx)
	if err != nil {
		return GetApiV1Users500JSONResponse{}, nil
	}

	ret := GetApiV1Users200JSONResponse{}
	for _, u := range users {
		ret = append(ret, User{
			Email: types.Email(u.Email),
		})
	}

	return ret, nil
}

/*
curl 'http://127.0.0.1:8050/api/v1/users/1' \
	--cookie-jar cookies.txt --cookie cookies.txt
*/

func (s *Server) GetApiV1UsersId(ctx context.Context, request GetApiV1UsersIdRequestObject) (GetApiV1UsersIdResponseObject, error) {
	user, err := db.New(s.db).GetUser(ctx, request.Id)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return GetApiV1UsersId404JSONResponse{}, nil
	}
	if err != nil {
		return GetApiV1UsersId500JSONResponse{}, nil
	}

	p := tpl.MainPage{
		BasePage: tpl.BasePage{CurrentUserName: fmt.Sprint(session.MustGetUser(ctx).Email)},
		Email:    user.Email,
	}

	return GetApiV1UsersId200TexthtmlResponse{Body: Body(func(w io.Writer) { tpl.WritePageTemplate(w, &p) })}, nil
}

func (s *Server) PostApiV1Users(ctx context.Context, request PostApiV1UsersRequestObject) (PostApiV1UsersResponseObject, error) {
	panic("not implemented")
}

func (s *Server) PutApiV1UsersId(ctx context.Context, request PutApiV1UsersIdRequestObject) (PutApiV1UsersIdResponseObject, error) {
	panic("not implemented")
}
