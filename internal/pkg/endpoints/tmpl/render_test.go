package tmpl

import (
	"context"
	"goweb/internal/pkg/oapi"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderPage(t *testing.T) {
	p := LoginPage(nil, "", oapi.Login{})
	renderer, _ := RenderPage[oapi.GetSessionsLogin200TexthtmlResponse](context.Background(), p)
	content, err := io.ReadAll(renderer.Body)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(content), 10)
}
