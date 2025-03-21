// Package oapi contains the OpenAPI spec and code generated from it.
package oapi

import (
	"context"
	_ "embed"
	"goweb/internal/pkg/logg"

	"github.com/getkin/kin-openapi/openapi3"
)

//go:generate go tool oapi-codegen -config oapi-cfg.yaml oapi.yaml

//go:embed oapi.yaml
var schemaBytes []byte

var schema *openapi3.T

func init() {
	var err error
	schema, err = openapi3.NewLoader().LoadFromData(schemaBytes)
	if err != nil {
		panic(err)
	}
}

// Schema gives access to the OpenAPI schema at runtime.
func Schema() *openapi3.T {
	return schema
}

// PrintRoutes logs all routes defined in the OpenAPI schema.
func PrintRoutes(ctx context.Context) {
	for path, item := range Schema().Paths.Map() {
		methods := []string{}
		for method := range item.Operations() {
			methods = append(methods, method)
		}
		logg.Debug(ctx, "Endpoint", "path", path, "methods", methods)
	}
}
