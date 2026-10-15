package ports

import (
	"context"
	"time"

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

// PeriodRepository persiste los períodos académicos (RF-03.2).
//
// Errores esperados: domain.ErrPeriodNotFound, domain.ErrPeriodCodeExists y
// domain.ErrPeriodClosed.
type PeriodRepository interface {
	// Create guarda el período y lo devuelve con su ID.
	Create(ctx context.Context, period domain.AcademicPeriod) (domain.AcademicPeriod, error)
	GetByID(ctx context.Context, id string) (domain.AcademicPeriod, error)
	// List devuelve todos los períodos, del que inicia más tarde al más antiguo.
	List(ctx context.Context) ([]domain.AcademicPeriod, error)
	// MarkClosed cierra el período de forma atómica: si ya estaba cerrado
	// devuelve domain.ErrPeriodClosed.
	MarkClosed(ctx context.Context, id string) error
}

// WorkspaceRepository persiste los cursos y proyectos (RF-03).
//
// Errores esperados: domain.ErrWorkspaceNotFound y domain.ErrWorkspaceClosed.
type WorkspaceRepository interface {
	// Create guarda el espacio y lo devuelve con su ID.
	Create(ctx context.Context, workspace domain.Workspace) (domain.Workspace, error)
	GetByID(ctx context.Context, id string) (domain.Workspace, error)
	List(ctx context.Context, filter WorkspaceFilter) (WorkspacePage, error)
	// MarkClosed cierra el espacio de forma atómica y registra cuándo: si ya
	// estaba cerrado devuelve domain.ErrWorkspaceClosed y no cambia su fecha de cierre.
	MarkClosed(ctx context.Context, id string, closedAt time.Time) error
}

// WorkspaceFilter delimita una consulta de espacios. Los campos vacíos no filtran.
type WorkspaceFilter struct {
	ProfessorID string
	PeriodID    string
	Status      domain.WorkspaceStatus
	Limit       int
	Offset      int
}

// WorkspacePage es una página de espacios con el total de coincidencias.
type WorkspacePage struct {
	Workspaces []domain.Workspace
	Total      int
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
