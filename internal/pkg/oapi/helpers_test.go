package oapi

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"jo-m.ch/go/goweb/internal/pkg/utl"
)

func TestMakeJSONError(t *testing.T) {
	ctx := context.WithValue(context.Background(), middleware.RequestIDKey, "1234")
	ret, _ := MakeJSONError[GetApiV1Users500JSONResponse](ctx)
	assert.Equal(t, GetApiV1Users500JSONResponse{
		N500InternalServerErrorJSONResponse: N500InternalServerErrorJSONResponse{
			Msg: "internal server error",
			On:  utl.Ptr("request"),
			Details: &[]ErrorJSON{
				{
					Msg: "1234",
					On:  utl.Ptr("reqID"),
				},
			},
		},
	}, ret)

	jsn, err := json.Marshal(ret)
	assert.NoError(t, err)
	assert.Equal(t, `{"details":[{"msg":"1234","on":"reqID"}],"msg":"internal server error","on":"request"}`, string(jsn))

	ret2, _ := MakeJSONError[GetApiV1UsersId404JSONResponse](context.Background())
	assert.Equal(t, GetApiV1UsersId404JSONResponse{
		N404NotFoundJSONResponse: N404NotFoundJSONResponse{
			Msg: "not found",
			On:  utl.Ptr("request"),
		},
	}, ret2)

	jsn, err = json.Marshal(ret2)
	assert.NoError(t, err)
	assert.Equal(t, `{"msg":"not found","on":"request"}`, string(jsn))
}
