package session

import (
	"context"
	"net/http"

	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/logg"
)

type ctxKeyUser struct{}

func withUser(ctx context.Context, user db.User) context.Context {
	return context.WithValue(ctx, ctxKeyUser{}, user)
}

// GetUser returns the user attached to a context,
// nil if there is none.
func GetUser(ctx context.Context) *db.User {
	if ret, ok := ctx.Value(ctxKeyUser{}).(db.User); ok {
		return &ret
	}
	return nil
}

// MustGetUser returns the user attached to a context
// and panics if there is none.
func MustGetUser(ctx context.Context) db.User {
	user := GetUser(ctx)
	if user != nil {
		return *user
	}

	logg.Panic(ctx, "No user attached to context")
	return db.User{} // unreachable
}

type ctxKeySession struct{}

func withSession(ctx context.Context, sess db.Session) context.Context {
	return context.WithValue(ctx, ctxKeySession{}, sess)
}

// Get returns the session attached to a context, or nil if there is none.
// This is the case for anonymous requests where no session cookie was sent.
func Get(ctx context.Context) *db.Session {
	if ret, ok := ctx.Value(ctxKeySession{}).(db.Session); ok {
		return &ret
	}
	return nil
}

// MustGet returns the session attached to a context
// and panics if there is none.
func MustGet(ctx context.Context) db.Session {
	if s := Get(ctx); s != nil {
		return *s
	}

	logg.Panic(ctx, "No session attached to context")
	return db.Session{} // unreachable
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

	logg.Panic(ctx, "No request attached to context")
	return requestCtx{} // unreachable
}
