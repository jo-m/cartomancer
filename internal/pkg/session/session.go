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
	"goweb/internal/pkg/logg"
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
	q *db.DB
	c Config
}

func NewStore(q *db.DB, c Config) Store {
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

func (s *Store) get(r *http.Request, tx *db.Queries) (*db.Session, error) {
	cookie, err := r.Cookie(s.c.CookieName)
	if err != nil {
		return nil, fmt.Errorf("no session id: %w", err)
	}

	parts := strings.SplitN(cookie.Value, ":", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid session string")
	}

	sess, err := tx.GetSession(r.Context(), parts[0])
	if err != nil {
		return nil, errors.New("no such session")
	}

	if !bytes.Equal(hash(parts[1]), sess.SecretHash) {
		return nil, errors.New("invalid secret")
	}

	now := time.Now()
	if now.Sub(sess.CreatedAt) > s.c.MaxAbsoluteTimeout {
		return nil, errors.New("session expired (absolute)")
	}

	if now.Sub(sess.LastActiveAt) > s.c.MaxIdleTimeout {
		return nil, errors.New("session expired (idle)")
	}

	err = tx.UpdateSessionLastActive(r.Context(), db.UpdateSessionLastActiveParams{
		ID:           sess.ID,
		LastActiveAt: now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update session last active: %w", err)
	}

	if sess.UserID.Valid {
		err := tx.UpdateUserLastActive(r.Context(), db.UpdateUserLastActiveParams{
			ID:           sess.UserID.String,
			LastActiveAt: sql.NullTime{Valid: true, Time: now},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update user last active: %w", err)
		}
	}

	return &sess, nil
}

func (s *Store) create(w http.ResponseWriter, r *http.Request, tx *db.Queries, userId sql.NullString, oldToDelete *db.Session) (*db.Session, error) {
	if oldToDelete != nil {
		err := tx.DeleteSession(r.Context(), oldToDelete.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to delete existing session: %w", err)
		}
	}

	// Update user last active.
	now := time.Now()
	if userId.Valid {
		err := tx.UpdateUserLastLogin(r.Context(), db.UpdateUserLastLoginParams{
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
		logg.Panic(r.Context(), "Secret hash length must be equal to sessionSecretBytes")
	}
	session, err := tx.CreateSession(r.Context(), params)
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

	tx, err := req.s.q.QueryTX(req.r.Context())
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	sess, err := req.s.create(req.w, req.r, tx, userId, oldToDelete)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return sess, nil
}

func (s *Store) delete(w http.ResponseWriter, r *http.Request, tx *db.Queries, sess *db.Session) error {
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
	http.SetCookie(w, &deleteCookie)

	return tx.DeleteSession(r.Context(), sess.ID)
}

// Delete a session.
// Ctx must be a request context which has passed through the session middleware.
func Delete(ctx context.Context, sess *db.Session) error {
	req := mustGetRequest(ctx)

	tx, err := req.s.q.QueryTX(req.r.Context())
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = req.s.delete(req.w, req.r, tx, sess)

	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) Middleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			tx, err := s.q.QueryTX(r.Context())
			if err != nil {
				logg.Error(r.Context(), "Failed to  begin transaction", "err", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer tx.Rollback()

			sess, err := s.get(r, tx)
			ctx := r.Context()
			if err != nil {
				logg.Debug(ctx, "No session found", "err", err)

				newSession, err := s.create(w, r, tx, sql.NullString{}, nil)
				if err != nil {
					logg.Error(ctx, "Failed to create session", "err", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				logg.Debug(ctx, "Created session", "id", newSession.ID)
				sess = newSession
			}
			if sess == nil {
				logg.Panic(ctx, "Session must not be nil at this point")
			}

			// Attach to context.
			logg.Debug(ctx, "Attaching session", "id", sess.ID)
			ctx = withSession(ctx, *sess)
			ctx = withRequest(ctx, requestCtx{w: w, r: r, s: s})

			// Fetch and attach user.
			if sess.UserID.Valid {
				user, err := tx.GetUser(ctx, sess.UserID.String)
				if err != nil {
					logg.Error(ctx, "Failed to retrieve user", "err", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				logg.Debug(ctx, "Attaching user", "id", user.ID)
				ctx = withUser(ctx, user)
			}

			err = tx.Commit()
			if err != nil {
				logg.Error(ctx, "Failed to commit", "err", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}
