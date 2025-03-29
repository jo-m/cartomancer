package oapi

import (
	"testing"

	"github.com/a-h/templ"
	"github.com/stretchr/testify/assert"
)

func TestLinksURL(t *testing.T) {
	links := Links{Base: "https://example.com"}
	assert.Equal(t, templ.SafeURL("https://example.com/api/v1/users"), links.GetAPIV1Users())
	assert.Equal(t, templ.SafeURL("https://example.com/api/v1/users/asdf"), links.GetAPIV1UsersID("asdf"))
	assert.Equal(t, templ.SafeURL("https://example.com/api/v1/users/asdf%2Fasdf"), links.GetAPIV1UsersID("asdf/asdf"))
	assert.Equal(t, templ.SafeURL("https://example.com/api/v1/users/asdf+asdf"), links.GetAPIV1UsersID("asdf asdf"))
}
