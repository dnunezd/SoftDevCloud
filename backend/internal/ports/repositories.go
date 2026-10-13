package ports

import (
	"context"

	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
)

// Repositorios de persistencia, uno por agregado. Cada módulo agrega aquí el
// suyo a medida que se implementa (períodos, espacios, vinculaciones, tareas,
// adjuntos y reportes). Implementación: internal/adapters/outbound/postgres.

// UserRepository persiste usuarios y sus roles (RF-01, RF-02).
//
// Errores esperados: domain.ErrUserNotFound y domain.ErrEmailAlreadyExists.
type UserRepository interface {
	// Create guarda el usuario con sus roles de forma atómica y lo devuelve con
	// su ID y fecha de creación.
	Create(ctx context.Context, user domain.User) (domain.User, error)
	GetByID(ctx context.Context, id string) (domain.User, error)
	// GetByEmail busca por el correo en su forma normalizada.
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	List(ctx context.Context, filter UserFilter) (UserPage, error)
	// ReplaceRoles reemplaza el conjunto completo de roles del usuario.
	ReplaceRoles(ctx context.Context, id string, roles []domain.Role) error
	CountWithRole(ctx context.Context, role domain.Role) (int, error)
}

// SessionRepository persiste las sesiones activas, una por inicio de sesión (DT-03).
//
// Error esperado: domain.ErrSessionNotFound.
type SessionRepository interface {
	// Create registra la sesión y, de paso, borra las sesiones vencidas del usuario.
	Create(ctx context.Context, session domain.Session) (domain.Session, error)
	// ActiveRoles devuelve los roles vigentes del dueño de una sesión activa (no
	// revocada ni vencida). Es la consulta que se ejecuta en cada petición
	// autenticada, para que un cierre de sesión o un cambio de roles apliquen de inmediato.
	ActiveRoles(ctx context.Context, sessionID, userID string) ([]domain.Role, error)
	// ListActive devuelve las sesiones activas del usuario, de la más reciente a la más antigua.
	ListActive(ctx context.Context, userID string) ([]domain.Session, error)
	// Revoke cierra una sesión. Es idempotente.
	Revoke(ctx context.Context, sessionID string) error
	// RevokeAll cierra todas las sesiones del usuario y devuelve cuántas eran.
	RevokeAll(ctx context.Context, userID string) (int, error)
	// DeleteExpired borra las sesiones vencidas de todos los usuarios.
	DeleteExpired(ctx context.Context) (int, error)
}

// UserFilter delimita una consulta de usuarios.
type UserFilter struct {
	// Role vacío significa "todos los roles".
	Role   domain.Role
	Limit  int
	Offset int
}

// UserPage es una página de usuarios con el total de coincidencias.
type UserPage struct {
	Users []domain.User
	Total int
}
