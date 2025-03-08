package session

import (
	"context"
	"goweb/internal/pkg/db"
	"goweb/internal/pkg/logging"
	"net/http"
)

type ctxKeyUser struct{}

func withUser(ctx context.Context, user db.User) context.Context {
	return context.WithValue(ctx, ctxKeyUser{}, user)
}

func GetUser(ctx context.Context) *db.User {
	if ret, ok := ctx.Value(ctxKeyUser{}).(db.User); ok {
		return &ret
	}
	return nil
}

func MustGetUser(ctx context.Context) db.User {
	user := GetUser(ctx)
	if user != nil {
		return *user
	}

	logging.Panic(ctx, "No user attached to context")
	panic("")
}

type ctxKeySession struct{}

func withSession(ctx context.Context, sess db.Session) context.Context {
	return context.WithValue(ctx, ctxKeySession{}, sess)
}

func MustGet(ctx context.Context) db.Session {
	if ret, ok := ctx.Value(ctxKeySession{}).(db.Session); ok {
		return ret
	}

	logging.Panic(ctx, "No session attached to context")
	panic("")
}

type ctxKeyRequest struct{}
type requestCtx struct {
	w http.ResponseWriter
	r *http.Request
	s *Store
}

// withRequest() is a bit of a hack to be able to access the request
// and response write in openapi-codegen strict server handlers.
func withRequest(ctx context.Context, req requestCtx) context.Context {
	return context.WithValue(
		ctx,
		ctxKeyRequest{},
		req,
	)
}

func mustGetRequest(ctx context.Context) requestCtx {
	if ret, ok := ctx.Value(ctxKeyRequest{}).(requestCtx); ok {
		return ret
	}

	logging.Panic(ctx, "No request attached to context")
	panic("")
}
