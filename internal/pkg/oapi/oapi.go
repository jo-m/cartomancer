// Package oapi contains the OpenAPI spec and code generated from it.
package oapi

import (
	_ "embed"

	"github.com/getkin/kin-openapi/openapi3"
)

//go:generate go tool oapi-codegen -config oapi-cfg.yaml oapi.yaml

//go:embed oapi.yaml
var schema []byte
var Schema *openapi3.T

func init() {
	var err error
	Schema, err = openapi3.NewLoader().LoadFromData(schema)
	if err != nil {
		panic(err)
	}
}
