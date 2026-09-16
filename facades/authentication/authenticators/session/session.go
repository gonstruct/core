// Package session is the database-backed authenticator behind
// config.SessionDriverDatabase.
//
// It hands the client an opaque random token and stores only its SHA-256, so a
// dump of the sessions table yields nothing a caller could present. That is the
// property a signed token cannot offer: because the row is the authority, a
// session can be revoked the moment it is deleted, rather than staying valid
// until it expires.
//
// The package speaks SQL rather than models so core stays independent of the
// generated model package. ParseSession is where the application turns a row
// back into whatever session type it uses, mirroring the jwt authenticator.
package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/gonstruct/core/facades/authentication"
	"time"

	"github.com/aarondl/sqlboiler/v4/boil"
)

// ErrSessionNotFound is returned when a token matches no live session, whether
// because it never existed or because it expired or was revoked. Callers must
// not distinguish those cases to the client.
var ErrSessionNotFound = errors.New("session not found")

const (
	tokenBytes      = 32
	defaultTable    = "sessions"
	defaultLifetime = 24 * time.Hour
)

// Details are the request facts recorded against a session so a user can review
// and revoke their own sessions later.
type Details struct {
	IP        string
	UserAgent string
}

type Service struct {
	// DB resolves the executor for session queries. It is a function because
	// providers register before the application binds its database, so the
	// executor cannot be captured at registration time. Required.
	DB func() boil.ContextExecutor

	// Lifetime is how long a new session stays valid. Defaults to 24h.
	Lifetime time.Duration

	// Table holds sessions. Defaults to "sessions".
	Table string

	// UserID extracts the owning user from a session being created.
	UserID func(session authentication.Session) string

	// ParseSession rebuilds the application's session type from a stored row.
	ParseSession func(id string, userID string) (authentication.Session, error)

	// Details supplies the ip and user agent for a new session. Optional.
	Details func() Details
}

func (s Service) GenerateToken(session authentication.Session) (authentication.Token, error) {
	plaintext, err := newToken()
	if err != nil {
		return nil, err
	}

	lifetime := session.Lifetime()
	if lifetime <= 0 {
		lifetime = s.lifetime()
	}

	details := Details{}
	if s.Details != nil {
		details = s.Details()
	}

	expiresAt := time.Now().UTC().Add(lifetime)

	var id string
	query := fmt.Sprintf(
		`INSERT INTO %s (user_id, token, ip, user_agent, expires_at) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		s.table(),
	)
	row := s.DB().QueryRowContext(
		context.Background(),
		query,
		s.UserID(session),
		hashToken(plaintext),
		details.IP,
		details.UserAgent,
		expiresAt,
	)
	if err := row.Scan(&id); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return sessionToken{id: id, plaintext: plaintext, expiresAt: expiresAt}, nil
}

func (s Service) ValidateToken(token string, _ ...bool) (authentication.Session, error) {
	if token == "" {
		return nil, ErrSessionNotFound
	}

	query := fmt.Sprintf(
		`SELECT id, user_id FROM %s WHERE token = $1 AND expires_at > NOW()`,
		s.table(),
	)

	var id, userID string
	err := s.DB().QueryRowContext(context.Background(), query, hashToken(token)).Scan(&id, &userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	// Touching last_activity_at is what makes an idle session distinguishable
	// from an abandoned one. A failure here must not fail the request: the
	// session is valid regardless of whether we recorded the visit.
	touch := fmt.Sprintf(`UPDATE %s SET last_activity_at = NOW(), updated_at = NOW() WHERE id = $1`, s.table())
	_, _ = s.DB().ExecContext(context.Background(), touch, id)

	return s.ParseSession(id, userID)
}

// Revoke deletes one session, ending it immediately.
func (s Service) Revoke(ctx context.Context, id string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, s.table())
	if _, err := s.DB().ExecContext(ctx, query, id); err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	return nil
}

// RevokeForUser ends every session a user holds, for a password change or a
// "sign out everywhere" action.
func (s Service) RevokeForUser(ctx context.Context, userID string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE user_id = $1`, s.table())
	if _, err := s.DB().ExecContext(ctx, query, userID); err != nil {
		return fmt.Errorf("failed to revoke sessions: %w", err)
	}

	return nil
}

// Prune removes expired rows. Expired sessions never authenticate, so this is
// housekeeping rather than a security boundary; run it on a schedule.
func (s Service) Prune(ctx context.Context) (int64, error) {
	query := fmt.Sprintf(`DELETE FROM %s WHERE expires_at <= NOW()`, s.table())
	result, err := s.DB().ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to prune sessions: %w", err)
	}

	return result.RowsAffected()
}

func (s Service) table() string {
	if s.Table == "" {
		return defaultTable
	}

	return s.Table
}

func (s Service) lifetime() time.Duration {
	if s.Lifetime <= 0 {
		return defaultLifetime
	}

	return s.Lifetime
}

func newToken() (string, error) {
	buffer := make([]byte, tokenBytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("failed to generate session token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))

	return hex.EncodeToString(sum[:])
}

type sessionToken struct {
	id        string
	plaintext string
	expiresAt time.Time
}

func (t sessionToken) ID() string {
	return t.id
}

func (t sessionToken) String() string {
	return t.plaintext
}

func (t sessionToken) ExpiresAt() time.Time {
	return t.expiresAt
}
