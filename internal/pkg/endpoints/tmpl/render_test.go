package tmpl

import (
	"io"
	"testing"

	"github.com/jo-m/goweb/internal/pkg/oapi"
	"github.com/stretchr/testify/assert"
)

func TestRenderPage(t *testing.T) {
	d := PageData{
		User: nil,
		L:    oapi.Links{},
	}

	p := LoginPage(d, "", oapi.Login{})
	renderer, _ := RenderPage[oapi.GetSessionsLogin200TexthtmlResponse](t.Context(), p)
	content, err := io.ReadAll(renderer.Body)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(content), 10)
}
