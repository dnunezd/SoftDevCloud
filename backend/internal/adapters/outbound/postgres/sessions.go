package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
)

const (
	insertSession = `
INSERT INTO sessions (user_id, user_agent, ip, expires_at) VALUES ($1, $2, $3, $4)
RETURNING id::text, created_at`
	deleteExpiredUserSessions = `DELETE FROM sessions WHERE user_id = $1 AND expires_at <= now()`
	deleteExpiredSessions     = `DELETE FROM sessions WHERE expires_at <= now()`
	selectActiveRoles         = `
SELECT COALESCE(array_agg(r.role ORDER BY r.role) FILTER (WHERE r.role IS NOT NULL), '{}')
FROM sessions s
LEFT JOIN user_roles r ON r.user_id = s.user_id
WHERE s.id = $1 AND s.user_id = $2 AND s.expires_at > now()
GROUP BY s.id`
	selectActiveSessions = `
SELECT id::text, user_id::text, user_agent, ip, created_at, expires_at
FROM sessions
WHERE user_id = $1 AND expires_at > now()
ORDER BY created_at DESC`
	deleteSession      = `DELETE FROM sessions WHERE id = $1`
	deleteUserSessions = `DELETE FROM sessions WHERE user_id = $1`
)

// SessionRepository implementa ports.SessionRepository sobre PostgreSQL.
type SessionRepository struct {
	pool *pgxpool.Pool
}

// NewSessionRepository crea el repositorio.
func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

// Create registra la sesión y borra en la misma transacción las sesiones
// vencidas del usuario, para que la tabla no crezca sin control.
func (r *SessionRepository) Create(ctx context.Context, s domain.Session) (domain.Session, error) {
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, deleteExpiredUserSessions, s.UserID); err != nil {
			return err
		}
		return tx.QueryRow(ctx, insertSession, s.UserID, s.UserAgent, s.IP, s.ExpiresAt).Scan(&s.ID, &s.CreatedAt)
	})
	if err != nil {
		return domain.Session{}, fmt.Errorf("crear sesión: %w", err)
	}
	return s, nil
}

// ActiveRoles devuelve los roles vigentes del dueño de una sesión activa, con
// una consulta por clave primaria. Se ejecuta en cada petición autenticada.
func (r *SessionRepository) ActiveRoles(ctx context.Context, sessionID, userID string) ([]domain.Role, error) {
	if !isUUID(sessionID) || !isUUID(userID) {
		return nil, domain.ErrSessionNotFound
	}
	var names []string
	err := r.pool.QueryRow(ctx, selectActiveRoles, sessionID, userID).Scan(&names)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("consultar sesión: %w", err)
	}
	return toRoles(names), nil
}

// ListActive devuelve las sesiones activas del usuario, de la más reciente a la más antigua.
func (r *SessionRepository) ListActive(ctx context.Context, userID string) ([]domain.Session, error) {
	if !isUUID(userID) {
		return []domain.Session{}, nil
	}
	rows, err := r.pool.Query(ctx, selectActiveSessions, userID)
	if err != nil {
		return nil, fmt.Errorf("listar sesiones: %w", err)
	}
	sessions, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Session, error) {
		var s domain.Session
		err := row.Scan(&s.ID, &s.UserID, &s.UserAgent, &s.IP, &s.CreatedAt, &s.ExpiresAt)
		return s, err
	})
	if err != nil {
		return nil, fmt.Errorf("leer sesiones: %w", err)
	}
	return sessions, nil
}

// Revoke cierra una sesión. Cerrar una sesión inexistente no es un error.
func (r *SessionRepository) Revoke(ctx context.Context, sessionID string) error {
	if !isUUID(sessionID) {
		return nil
	}
	if _, err := r.pool.Exec(ctx, deleteSession, sessionID); err != nil {
		return fmt.Errorf("cerrar sesión: %w", err)
	}
	return nil
}

// RevokeAll cierra todas las sesiones del usuario y devuelve cuántas eran.
func (r *SessionRepository) RevokeAll(ctx context.Context, userID string) (int, error) {
	if !isUUID(userID) {
		return 0, nil
	}
	tag, err := r.pool.Exec(ctx, deleteUserSessions, userID)
	if err != nil {
		return 0, fmt.Errorf("cerrar sesiones: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// DeleteExpired borra las sesiones vencidas de todos los usuarios.
func (r *SessionRepository) DeleteExpired(ctx context.Context) (int, error) {
	tag, err := r.pool.Exec(ctx, deleteExpiredSessions)
	if err != nil {
		return 0, fmt.Errorf("borrar sesiones vencidas: %w", err)
	}
	return int(tag.RowsAffected()), nil
}
