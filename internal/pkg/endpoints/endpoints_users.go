package endpoints

//lint:file-ignore ST1020,ST1003 Ignore generated method names.

import (
	"context"
	"database/sql"
	"errors"
	"goweb/internal/pkg/endpoints/tpl"
	"goweb/internal/pkg/oapi"
	"goweb/internal/pkg/session"
	"io"

	"github.com/oapi-codegen/runtime/types"
)

/*
	curl -v 'http://127.0.0.1:8050/api/v1/users' \
		--cookie-jar cookies.txt --cookie cookies.txt
*/
func (s *Server) GetApiV1Users(ctx context.Context, request oapi.GetApiV1UsersRequestObject) (oapi.GetApiV1UsersResponseObject, error) {
	users, err := s.d.QueryRO().GetUsers(ctx)
	if err != nil {
		return oapi.GetApiV1Users500JSONResponse{}, nil
	}

	ret := oapi.GetApiV1Users200JSONResponse{}
	for _, u := range users {
		ret = append(ret, oapi.User{
			Email: types.Email(u.Email),
		})
	}

	return ret, nil
}

/*
curl 'http://127.0.0.1:8050/api/v1/users/1' \
	--cookie-jar cookies.txt --cookie cookies.txt
*/

func (s *Server) GetApiV1UsersId(ctx context.Context, request oapi.GetApiV1UsersIdRequestObject) (oapi.GetApiV1UsersIdResponseObject, error) {
	user, err := s.d.QueryRO().GetUser(ctx, request.Id)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return oapi.GetApiV1UsersId404JSONResponse{}, nil
	}
	if err != nil {
		return oapi.GetApiV1UsersId500JSONResponse{}, nil
	}

	p := tpl.MainPage{
		BasePage: tpl.BasePage{User: session.GetUser(ctx)},
		Email:    user.Email,
	}

	return oapi.GetApiV1UsersId200TexthtmlResponse{Body: Body(func(w io.Writer) { tpl.WritePageTemplate(w, &p) })}, nil
}

func (s *Server) PostApiV1Users(ctx context.Context, request oapi.PostApiV1UsersRequestObject) (oapi.PostApiV1UsersResponseObject, error) {
	panic("not implemented")
}

func (s *Server) PutApiV1UsersId(ctx context.Context, request oapi.PutApiV1UsersIdRequestObject) (oapi.PutApiV1UsersIdResponseObject, error) {
	panic("not implemented")
}
