// Package session deals with sessions.
package session

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"goweb/internal/pkg/app"
	"goweb/internal/pkg/db"
	"goweb/internal/pkg/logg"
	"goweb/internal/pkg/password"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
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
	JWTSecret    string `arg:"--session-jwt-secret,env:SESSION_JWT_SECRET" help:"Secret to sign JWT, generated on startup if not set" placeholder:"SECRET"`
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
		d: d,
		c: c,
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
	claims, err := jwtParseAndVerify(cookie.Value, now, []byte(s.c.JWTSecret), s.ac.AppName)
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
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate uuid: %w", err)
	}
	params := db.CreateSessionParams{
		ID:           id.String(),
		CreatedAt:    now,
		LastActiveAt: now,
		UserID:       userID,
	}
	session, err := tx.CreateSession(r.Context(), params)
	if err != nil {
		return nil, fmt.Errorf("failed to create session in db: %w", err)
	}

	claims := claimsForSession(id.String(), now, s.c.AbsoluteTimeout, s.ac.AppName)
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
