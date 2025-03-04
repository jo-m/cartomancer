package session

import (
	"context"
	"goweb/internal/pkg/db"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	ctx := context.Background()

	sess := db.Session{ID: "asdf"}
	ctx = withSession(ctx, sess)
	assert.Equal(t, sess, MustGetSession(ctx))

	user := db.User{ID: "asdf"}
	ctx = withUser(ctx, user)
	assert.Equal(t, user, MustGetUser(ctx))
}

func TestHash(t *testing.T) {
	s := "asdf"
	assert.Equal(t, len(hash(s)), sessionSecretBytes, "Secret hash length must be equal to sessionSecretBytes")
}
