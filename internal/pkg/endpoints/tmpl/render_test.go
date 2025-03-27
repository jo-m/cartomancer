package tmpl

import (
	"goweb/internal/pkg/oapi"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderPage(t *testing.T) {
	pCtx := PageContext{
		User: nil,
		L:    oapi.Links{},
	}

	p := LoginPage(pCtx, "", oapi.Login{})
	renderer, _ := RenderPage[oapi.GetSessionsLogin200TexthtmlResponse](t.Context(), p)
	content, err := io.ReadAll(renderer.Body)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(content), 10)
}
