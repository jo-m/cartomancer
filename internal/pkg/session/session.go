// Package session deals with sessions.
package session

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"jo-m.ch/go/detour/internal/pkg/app"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/logg"
	"jo-m.ch/go/detour/internal/pkg/password"
)

// SessionConfig options for a session store.
// It has struct tags compatible with [github.com/alexflint/go-arg].
//
//revive:disable:exported Naming necessary for struct embedding.
type SessionConfig struct {
	IdleTimeout     time.Duration `arg:"--session-idle-timeout,env:SESSION_IDLE_TIMEOUT" default:"24h" help:"Session idle timeout" placeholder:"DUR"`
	AbsoluteTimeout time.Duration `arg:"--session-abs-timeout,env:SESSION_ABS_TIMEOUT" default:"72h" help:"Session absolute timeout" placeholder:"DUR"`
	CookieName      string        `arg:"--session-cookie-name,env:SESSION_COOKIE_NAME" default:"sid" help:"Session cookie name" placeholder:"NAME"`
	// REQUIRED for production deployments.
	JWTSecret    string `arg:"--session-jwt-secret,env:SESSION_JWT_SECRET" help:"Secret to sign JWT, generated on startup if not set" placeholder:"SECRET"`
	CookieDomain string `arg:"--session-cookie-domain,env:SESSION_COOKIE_DOMAIN" default:"" help:"Session cookie domain" placeholder:"HOST"`
	CookiePath   string `arg:"--session-cookie-path,env:SESSION_COOKIE_PATH" default:"/" help:"Session cookie path" placeholder:"PATH"`
	// Default is safe.
	insecureUseOnlyForTests bool
}

// Validate checks for basic configuration errors.
func (c *SessionConfig) Validate() error {
	if c.IdleTimeout <= 0 {
		return errors.New("--session-idle-timeout / SESSION_IDLE_TIMEOUT must be positive")
	}
	if c.AbsoluteTimeout <= 0 {
		return errors.New("--session-abs-timeout / SESSION_ABS_TIMEOUT must be positive")
	}
	if c.IdleTimeout > c.AbsoluteTimeout {
		return errors.New("--session-idle-timeout / SESSION_IDLE_TIMEOUT must not exceed --session-abs-timeout / SESSION_ABS_TIMEOUT")
	}
	if c.CookieName == "" {
		return errors.New("--session-cookie-name / SESSION_COOKIE_NAME must not be empty")
	}

	return nil
}

// ValidateProduction checks that all settings required for a production deployment are set.
func (c *SessionConfig) ValidateProduction() error {
	if c.JWTSecret == "" {
		return errors.New("--session-jwt-secret / SESSION_JWT_SECRET is required for production")
	}
	if c.CookieDomain == "" {
		return errors.New("--session-cookie-domain / SESSION_COOKIE_DOMAIN is required for production")
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
// Use [NewStore] to create an instance.
type Store struct {
	d  *db.DB
	c  SessionConfig
	ac app.AppConfig
}

// NewStore creates a new session store.
func NewStore(d *db.DB, c SessionConfig, ac app.AppConfig) (*Store, error) {
	err := c.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid session config: %w", err)
	}
	if len(c.JWTSecret) == 0 {
		c.JWTSecret = string(password.GenRandBytes(jwtSecretLenBytes))
	}
	if len(c.JWTSecret) != jwtSecretLenBytes {
		return nil, fmt.Errorf("JWT secret must be %d bytes but is %d", jwtSecretLenBytes, len(c.JWTSecret))
	}

	return &Store{
		d:  d,
		c:  c,
		ac: ac,
	}, nil
}

var (
	ErrNoSuchSession          = errors.New("no such session")
	ErrSessionExpiredIdle     = errors.New("session expired (idle)")
	ErrSessionExpiredAbsolute = errors.New("session expired (absolute)")
)

func (s *Store) get(r *http.Request, tx *db.Queries) (*db.Session, error) {
	cookie, err := r.Cookie(s.c.CookieName)
	if err != nil {
		return nil, fmt.Errorf("missing session id: %w", err)
	}

	now := time.Now()
	claims, err := jwtParseAndVerify(cookie.Value, now, []byte(s.c.JWTSecret), s.ac.InstanceName)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrSessionExpiredAbsolute
		}

		return nil, fmt.Errorf("invalid JWT: %w", err)
	}

	sess, err := tx.GetSession(r.Context(), claims.ID)
	if err != nil {
		return nil, ErrNoSuchSession
	}

	if now.Sub(sess.CreatedAt) > s.c.AbsoluteTimeout {
		return nil, ErrSessionExpiredAbsolute
	}

	if now.Sub(sess.LastActiveAt) > s.c.IdleTimeout {
		return nil, ErrSessionExpiredIdle
	}

	err = db.EnsureOneRowChanged(tx.UpdateSessionLastActive(r.Context(), db.UpdateSessionLastActiveParams{
		Uuid:         sess.Uuid,
		LastActiveAt: now,
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to update session last active: %w", err)
	}

	if sess.UserID.Valid {
		err := db.EnsureOneRowChanged(tx.UpdateUserLastActive(r.Context(), db.UpdateUserLastActiveParams{
			Uuid:         sess.UserID.String,
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
		err := db.EnsureOneRowChanged(tx.DeleteSession(r.Context(), oldToDelete.Uuid))
		if err != nil {
			return nil, fmt.Errorf("failed to delete existing session: %w", err)
		}
	}

	// Update user last active.
	now := time.Now()
	if userID.Valid {
		err := db.EnsureOneRowChanged(tx.UpdateUserLastLogin(r.Context(), db.UpdateUserLastLoginParams{
			Uuid:         userID.String,
			LastLoginAt:  sql.NullTime{Valid: true, Time: now},
			LastActiveAt: sql.NullTime{Valid: true, Time: now},
		}))
		if err != nil {
			return nil, fmt.Errorf("failed to set user last active: %w", err)
		}
	}

	// Create new.
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate uuid: %w", err)
	}
	params := db.CreateSessionParams{
		Uuid:         id.String(),
		CreatedAt:    now,
		LastActiveAt: now,
		UserID:       userID,
	}
	session, err := tx.CreateSession(r.Context(), params)
	if err != nil {
		return nil, fmt.Errorf("failed to create session in db: %w", err)
	}

	claims := claimsForSession(id.String(), now, s.c.AbsoluteTimeout, s.ac.InstanceName)
	token, err := jwtSign(claims, []byte(s.c.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign JWT: %w", err)
	}

	// And send the cookie.
	cookie := http.Cookie{
		Name:     s.c.CookieName,
		Value:    token,
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

	return db.EnsureOneRowChanged(tx.DeleteSession(r.Context(), sess.Uuid))
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

// Middleware loads existing sessions from cookies and attaches session and user
// info to the request context. Anonymous requests without a valid session cookie
// proceed without a session; handlers that need one can call [Create] on demand.
func (s *Store) Middleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Always attach request context so Create/Delete work from handlers.
		ctx = withRequest(ctx, requestCtx{w: w, r: r, s: s})

		tx, err := s.d.BeginTX(ctx)
		if err != nil {
			logg.Error(ctx, "Failed to begin transaction", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		sess, err := s.get(r, tx)
		if err != nil {
			logg.Debug(ctx, "No session found", "err", err)
		}

		if sess != nil {
			logg.Debug(ctx, "Attaching session", "id", sess.Uuid)
			ctx = withSession(ctx, *sess)

			// Fetch and attach user.
			if sess.UserID.Valid {
				user, err := tx.GetUser(ctx, sess.UserID.String)
				if err != nil {
					logg.Error(ctx, "Failed to retrieve user", "err", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				logg.Debug(ctx, "Attaching user", "id", user.Uuid)
				ctx = withUser(ctx, user)
			}
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
