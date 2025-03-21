package oapi

import (
	"goweb/internal/pkg/utl"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeJSONError(t *testing.T) {
	ret, _ := MakeJSONError[GetApiV1Users500JSONResponse]()
	assert.Equal(t, GetApiV1Users500JSONResponse{
		N500InternalServerErrorJSONResponse: N500InternalServerErrorJSONResponse{
			Error: "internal server error",
			Scope: utl.Ptr("request"),
		},
	}, ret)

	ret2, _ := MakeJSONError[GetApiV1UsersId404JSONResponse]()
	assert.Equal(t, GetApiV1UsersId404JSONResponse{
		N404NotFoundJSONResponse: N404NotFoundJSONResponse{
			Error: "not found",
			Scope: utl.Ptr("request"),
		},
	}, ret2)
}
