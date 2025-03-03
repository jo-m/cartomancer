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
	"goweb/internal/pkg/password"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// TODO: replace slog with logger

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
	sessionSecretBytes = 64
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

type ctxKey struct{}

func Middleware(c Config) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			session, err := c.retrieveSession(r)
			if err != nil {
				slog.Debug("No session found", "err", err)

				newSession, err := c.createSession(w, r)
				if err != nil {
					slog.Error("Failed to create session", "err", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				slog.Debug("Created session", "id", newSession.ID)
				session = newSession
			}

			if session == nil {
				slog.Error("Session is nil")
				panic("session is nil")
			}

			slog.Debug("Attaching session", "id", session.ID)
			ctx := context.WithValue(r.Context(), ctxKey{}, session.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}

func GetSessionID(ctx context.Context) int64 {
	if ret, ok := ctx.Value(ctxKey{}).(int64); ok {
		return ret
	}

	panic("no session ID attached to context")
}
