package endpoints

//lint:file-ignore ST1020,ST1003 Ignore generated method names.

import (
	"context"
	"database/sql"
	"errors"
	"goweb/internal/pkg/logg"
	"goweb/internal/pkg/oapi"
	"goweb/internal/pkg/utl"

	"github.com/oapi-codegen/runtime/types"
)

/*
	curl -v 'http://127.0.0.1:8050/api/v1/users' \
		--compressed \
		--cookie-jar cookies.txt --cookie cookies.txt
*/
func (s *Server) GetApiV1Users(ctx context.Context, request oapi.GetApiV1UsersRequestObject) (oapi.GetApiV1UsersResponseObject, error) {
	users, err := s.d.QueryRO().GetUsers(ctx)
	if err != nil {
		logg.Error(ctx, "Failed to query", "err", err)
		return mk500[oapi.GetApiV1Users500JSONResponse](), nil
	}

	ret := oapi.GetApiV1Users200JSONResponse{}
	for _, u := range users {
		ret = append(ret, oapi.User{
			Id:    utl.Ptr(u.ID),
			Email: types.Email(u.Email),
			Name:  u.Name,
		})
	}

	return ret, nil
}

/*
	curl -v 'http://127.0.0.1:8050/api/v1/users/123' \
		--compressed \
		--cookie-jar cookies.txt --cookie cookies.txt
*/
func (s *Server) GetApiV1UsersId(ctx context.Context, request oapi.GetApiV1UsersIdRequestObject) (oapi.GetApiV1UsersIdResponseObject, error) {
	user, err := s.d.QueryRO().GetUser(ctx, request.Id)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return mk404[oapi.GetApiV1UsersId404JSONResponse](), nil
	}
	if err != nil {
		return mk500[oapi.GetApiV1UsersId500JSONResponse](), nil
	}

	return oapi.GetApiV1UsersId200JSONResponse{
		Id:    utl.Ptr(user.ID),
		Email: types.Email(user.Email),
		Name:  user.Name,
	}, nil
}

func (s *Server) PostApiV1Users(ctx context.Context, request oapi.PostApiV1UsersRequestObject) (oapi.PostApiV1UsersResponseObject, error) {
	panic("not implemented")
}

func (s *Server) PutApiV1UsersId(ctx context.Context, request oapi.PutApiV1UsersIdRequestObject) (oapi.PutApiV1UsersIdResponseObject, error) {
	panic("not implemented")
}
