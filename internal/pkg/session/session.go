package session

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"goweb/internal/pkg/db"
	"goweb/internal/pkg/logging"
	"goweb/internal/pkg/password"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DB         *sql.DB
	MaxAge     time.Duration
	CookieName string
	CookiePath string
}

func (c *Config) q() *db.Queries {
	return db.New(c.DB)
}

const (
	// 64 bits are recommended, we have 32 * 8 = 256 bits here.
	// https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html#session-id-entropy
	sessionSecretBytes = 32
)

func hash(s string) []byte {
	h := sha256.New()
	h.Write([]byte(s))
	return h.Sum(nil)
}

func (c *Config) retrieveSession(r *http.Request) (*db.Session, error) {
	cookie, err := r.Cookie(c.CookieName)
	if err != nil {
		return nil, fmt.Errorf("no session id: %w", err)
	}

	parts := strings.SplitN(cookie.Value, ":", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid session string")
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, errors.New("invalid session id")
	}

	session, err := c.q().GetSession(r.Context(), id)
	if err != nil {
		return nil, errors.New("no such session")
	}

	if time.Since(session.CreatedAt) > time.Minute*30 {
		return nil, errors.New("session expired")
	}

	if !bytes.Equal(hash(parts[1]), session.SecretHash) {
		return nil, errors.New("invalid secret")
	}

	return &session, nil
}

func (c *Config) createSession(w http.ResponseWriter, r *http.Request) (*db.Session, error) {
	secretBytes := password.GenRandBytes(sessionSecretBytes)
	secret := base64.URLEncoding.EncodeToString(secretBytes)
	now := time.Now()
	params := db.CreateSessionParams{
		CreatedAt:  now,
		ExpiresAt:  now.Add(c.MaxAge),
		SecretHash: hash(secret),
	}
	if len(params.SecretHash) != sessionSecretBytes {
		logging.Panic(r.Context(), "Secret hash length must be equal to sessionSecretBytes")
	}
	session, err := c.q().CreateSession(r.Context(), params)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	cookie := http.Cookie{
		Name:     c.CookieName,
		Value:    fmt.Sprintf("%d:%s", session.ID, secret),
		MaxAge:   int(c.MaxAge.Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     c.CookiePath,
	}
	http.SetCookie(w, &cookie)

	return &session, nil
}

type ctxKeySession struct{}
type ctxKeyUser struct{}

func Middleware(c Config) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			session, err := c.retrieveSession(r)
			ctx := r.Context()
			if err != nil {
				logging.Debug(ctx, "No session found", "err", err)

				newSession, err := c.createSession(w, r)
				if err != nil {
					logging.Error(ctx, "Failed to create session", "err", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				logging.Debug(ctx, "Created session", "id", newSession.ID)
				session = newSession
			}

			if session == nil {
				logging.Panic(ctx, "Session must not be nil")
			}

			logging.Debug(ctx, "Attaching session", "id", session.ID)
			ctx = withSession(ctx, *session)

			if session.UserID.Valid {
				user, err := c.q().GetUser(ctx, session.UserID.Int64)
				if err != nil {
					logging.Error(ctx, "Failed to retrieve user", "err", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				logging.Debug(ctx, "Attaching user", "id", user.ID)
				ctx = withUser(ctx, user)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}

func withSession(ctx context.Context, sess db.Session) context.Context {
	return context.WithValue(ctx, ctxKeySession{}, sess)
}

func MustGetSession(ctx context.Context) db.Session {
	if ret, ok := ctx.Value(ctxKeySession{}).(db.Session); ok {
		return ret
	}

	logging.Panic(ctx, "No session attached to context")
	panic("")
}

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
