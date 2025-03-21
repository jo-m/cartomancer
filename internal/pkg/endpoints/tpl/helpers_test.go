package tpl

import (
	"goweb/internal/pkg/oapi"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderPage(t *testing.T) {
	p := LoginPage{}
	renderer, _ := RenderPage[oapi.PostSessionsLogin401TexthtmlResponse](&p)
	content, err := io.ReadAll(renderer.Body)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(content), 10)
}
