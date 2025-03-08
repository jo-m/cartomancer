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
	"strings"
	"time"

	"github.com/google/uuid"
)

type Config struct {
	MaxIdleTimeout     time.Duration
	MaxAbsoluteTimeout time.Duration
	CookieName         string
	CookieDomain       string
	CookiePath         string
}

type Store struct {
	q *db.Queries
	c Config
}

func NewStore(q *db.Queries, c Config) Store {
	return Store{
		q: q,
		c: c,
	}
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

func (s *Store) get(r *http.Request) (*db.Session, error) {
	cookie, err := r.Cookie(s.c.CookieName)
	if err != nil {
		return nil, fmt.Errorf("no session id: %w", err)
	}

	parts := strings.SplitN(cookie.Value, ":", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid session string")
	}

	session, err := s.q.GetSession(r.Context(), parts[0])
	if err != nil {
		return nil, errors.New("no such session")
	}

	if !bytes.Equal(hash(parts[1]), session.SecretHash) {
		return nil, errors.New("invalid secret")
	}

	now := time.Now()
	if now.Sub(session.CreatedAt) > s.c.MaxAbsoluteTimeout {
		return nil, errors.New("session expired (absolute)")
	}

	if now.Sub(session.LastActiveAt) > s.c.MaxIdleTimeout {
		return nil, errors.New("session expired (idle)")
	}

	err = s.q.UpdateSessionLastActive(r.Context(), db.UpdateSessionLastActiveParams{
		ID:           session.ID,
		LastActiveAt: now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update session last active: %w", err)
	}

	if session.UserID.Valid {
		err := s.q.UpdateUserLastActive(r.Context(), db.UpdateUserLastActiveParams{
			ID:           session.UserID.String,
			LastActiveAt: sql.NullTime{Valid: true, Time: now},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update user last active: %w", err)
		}
	}

	return &session, nil
}

func (s *Store) create(w http.ResponseWriter, r *http.Request, userId sql.NullString, oldToDelete *db.Session) (*db.Session, error) {
	if oldToDelete != nil {
		err := s.q.DeleteSession(r.Context(), oldToDelete.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to delete existing session: %w", err)
		}
	}

	// Update user last active.
	now := time.Now()
	if userId.Valid {
		err := s.q.UpdateUserLastLogin(r.Context(), db.UpdateUserLastLoginParams{
			ID:           userId.String,
			LastLoginAt:  sql.NullTime{Valid: true, Time: now},
			LastActiveAt: sql.NullTime{Valid: true, Time: now},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to set user last active: %w", err)
		}
	}

	// Create new.
	secretBytes := password.GenRandBytes(sessionSecretBytes)
	secret := base64.URLEncoding.EncodeToString(secretBytes)
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate uuid: %w", err)
	}
	params := db.CreateSessionParams{
		ID:           id.String(),
		CreatedAt:    now,
		LastActiveAt: now,
		SecretHash:   hash(secret),
		UserID:       userId,
	}
	if len(params.SecretHash) != sessionSecretBytes {
		logging.Panic(r.Context(), "Secret hash length must be equal to sessionSecretBytes")
	}
	session, err := s.q.CreateSession(r.Context(), params)
	if err != nil {
		return nil, fmt.Errorf("failed to create session in db: %w", err)
	}

	// And send the cookie.
	cookie := http.Cookie{
		Name:     s.c.CookieName,
		Value:    fmt.Sprintf("%s:%s", session.ID, secret),
		MaxAge:   int(s.c.MaxAbsoluteTimeout.Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     s.c.CookiePath,
		Domain:   s.c.CookieDomain,
	}
	http.SetCookie(w, &cookie)

	return &session, nil
}

// Create a session.
// Ctx must be a request context which has passed through the session middleware.
func Create(ctx context.Context, userId sql.NullString, oldToDelete *db.Session) (*db.Session, error) {
	req := mustGetRequest(ctx)
	return req.s.create(req.w, req.r, userId, oldToDelete)
}

func (s *Store) delete(req requestCtx, session *db.Session) error {
	deleteCookie := http.Cookie{
		Name:     s.c.CookieName,
		Value:    "",
		MaxAge:   0,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     s.c.CookiePath,
		Domain:   s.c.CookieDomain,
	}
	http.SetCookie(req.w, &deleteCookie)

	return s.q.DeleteSession(req.r.Context(), session.ID)
}

// Delete a session.
// Ctx must be a request context which has passed through the session middleware.
func Delete(ctx context.Context, session *db.Session) error {
	req := mustGetRequest(ctx)
	return req.s.delete(req, session)
}

func (s *Store) Middleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			session, err := s.get(r)
			ctx := r.Context()
			if err != nil {
				logging.Debug(ctx, "No session found", "err", err)

				newSession, err := s.create(w, r, sql.NullString{}, nil)
				if err != nil {
					logging.Error(ctx, "Failed to create session", "err", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				logging.Debug(ctx, "Created session", "id", newSession.ID)
				session = newSession
			}
			if session == nil {
				logging.Panic(ctx, "Session must not be nil at this point")
			}

			// Attach to context.
			logging.Debug(ctx, "Attaching session", "id", session.ID)
			ctx = withSession(ctx, *session)
			ctx = withRequest(ctx, requestCtx{w: w, r: r, s: s})

			// Fetch and attach user.
			if session.UserID.Valid {
				user, err := s.q.GetUser(ctx, session.UserID.String)
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
