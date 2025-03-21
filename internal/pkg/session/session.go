// Package session deals with sessions.
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

// SessionConfig options for a session store.
// It has struct tags compatible with github.com/alexflint/go-arg.
//
//revive:disable:exported Naming necessary for struct embedding.
type SessionConfig struct {
	// Required.
	IdleTimeout     time.Duration `arg:"--session-idle-timeout,env:SESSION_IDLE_TIMEOUT" default:"0" help:"Session idle timeout" placeholder:"DUR"`
	AbsoluteTimeout time.Duration `arg:"--session-abs-timeout,env:SESSION_ABS_TIMEOUT" default:"0" help:"Session absolute timeout" placeholder:"DUR"`
	CookieName      string        `arg:"--session-cookie-name,env:SESSION_COOKIE_NAME" default:"" help:"Session cookie name" placeholder:"NAME"`
	// Optional.
	CookieDomain string `arg:"--session-cookie-domain,env:SESSION_COOKIE_DOMAIN" default:"" help:"Session cookie domain" placeholder:"HOST"`
	CookiePath   string `arg:"--session-cookie-path,env:SESSION_COOKIE_PATH" default:"/" help:"Session cookie path" placeholder:"PATH"`
	// Default is safe.
	insecureUseOnlyForTests bool
}

// Validate the session config.
func (c *SessionConfig) Validate() error {
	if c.IdleTimeout <= 0 {
		return errors.New("idle timeout must be positive")
	}
	if c.AbsoluteTimeout <= 0 {
		return errors.New("absolute timeout must be positive")
	}
	if c.IdleTimeout > c.AbsoluteTimeout {
		return errors.New("absolute timeout must be greater or equal absolute timeout")
	}
	if c.CookieName == "" {
		return errors.New("cookie name must not be empty")
	}
	return nil
}

// GetCleanerArgs returns the arguments for the session cleaner job.
//
//revive:disable:unexported-return
func (c *SessionConfig) GetCleanerArgs() cleanerArgs {
	return cleanerArgs{
		IdleTimeout:     c.IdleTimeout,
		AbsoluteTimeout: c.AbsoluteTimeout,
	}
}

// Store manages sessions.
// Use NewStore() to create an instance.
type Store struct {
	d *db.DB
	c SessionConfig
}

// NewStore creates a new session store.
func NewStore(d *db.DB, c SessionConfig) (*Store, error) {
	err := c.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid session config: %w", err)
	}

	return &Store{
		d: d,
		c: c,
	}, nil
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
	if now.Sub(sess.CreatedAt) > s.c.AbsoluteTimeout {
		return nil, errors.New("session expired (absolute)")
	}

	if now.Sub(sess.LastActiveAt) > s.c.IdleTimeout {
		return nil, errors.New("session expired (idle)")
	}

	err = db.EnsureOneRowChanged(tx.UpdateSessionLastActive(r.Context(), db.UpdateSessionLastActiveParams{
		ID:           sess.ID,
		LastActiveAt: now,
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to update session last active: %w", err)
	}

	if sess.UserID.Valid {
		err := db.EnsureOneRowChanged(tx.UpdateUserLastActive(r.Context(), db.UpdateUserLastActiveParams{
			ID:           sess.UserID.String,
			LastActiveAt: sql.NullTime{Valid: true, Time: now},
		}))
		if err != nil {
			return nil, fmt.Errorf("failed to update user last active: %w", err)
		}
	}

	return &sess, nil
}

func (s *Store) create(w http.ResponseWriter, r *http.Request, tx *db.Queries, userID sql.NullString, oldToDelete *db.Session) (*db.Session, error) {
	if oldToDelete != nil {
		err := db.EnsureOneRowChanged(tx.DeleteSession(r.Context(), oldToDelete.ID))
		if err != nil {
			return nil, fmt.Errorf("failed to delete existing session: %w", err)
		}
	}

	// Update user last active.
	now := time.Now()
	if userID.Valid {
		err := db.EnsureOneRowChanged(tx.UpdateUserLastLogin(r.Context(), db.UpdateUserLastLoginParams{
			ID:           userID.String,
			LastLoginAt:  sql.NullTime{Valid: true, Time: now},
			LastActiveAt: sql.NullTime{Valid: true, Time: now},
		}))
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
		UserID:       userID,
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
		MaxAge:   int(s.c.AbsoluteTimeout.Seconds()),
		Secure:   !s.c.insecureUseOnlyForTests,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     s.c.CookiePath,
		Domain:   s.c.CookieDomain,
	}
	http.SetCookie(w, &cookie)

	return &session, nil
}

// Create a session in the database and set the session cookie.
// Ctx must be a request context which has previously passed
// through the session middleware, so that the required items have been attached to it.
func Create(ctx context.Context, userID sql.NullString, oldToDelete *db.Session) (*db.Session, error) {
	req := mustGetRequest(ctx)

	var sess *db.Session
	err := req.s.d.WithTx(ctx, func(tx *db.Queries) error {
		var err error
		sess, err = req.s.create(req.w, req.r, tx, userID, oldToDelete)
		return err
	})
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
		Secure:   !s.c.insecureUseOnlyForTests,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     s.c.CookiePath,
		Domain:   s.c.CookieDomain,
	}
	http.SetCookie(w, &deleteCookie)

	return db.EnsureOneRowChanged(tx.DeleteSession(r.Context(), sess.ID))
}

// Delete a session from the database and delete the session cookie.
// Ctx must be a request context which has previously passed
// through the session middleware, so that the required items have been attached to it.
func Delete(ctx context.Context, sess *db.Session) error {
	req := mustGetRequest(ctx)

	return req.s.d.WithTx(ctx, func(tx *db.Queries) error {
		return req.s.delete(req.w, req.r, tx, sess)
	})
}

func (s *Store) cleanup(ctx context.Context, now time.Time) error {
	_, err := s.d.QueryRW().CleanupSessions(ctx, db.CleanupSessionsParams{
		CreatedBefore: now.Add(-s.c.AbsoluteTimeout),
		ActiveBefore:  now.Add(-s.c.IdleTimeout),
	})
	return err
}

// Middleware automatically issues sessions for each request,
// sends the session token to the user as cookie,
// and attaches session and user info to the request context.
func (s *Store) Middleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		tx, err := s.d.BeginTX(r.Context())
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
