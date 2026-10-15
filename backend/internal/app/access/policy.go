// Package access concentra las políticas de autorización de grano fino.
//
// Reglas: el profesor solo ve a sus propios monitores y asistentes; el usuario
// operativo solo ve sus propios registros; el administrador ve todo
// (RF-12, RN-03 a RN-05). El middleware HTTP valida roles; este paquete valida
// la relación entre el usuario y el dato.
package access

import (
	"slices"

	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
)

// Principal es la identidad autenticada que realiza una operación: el usuario,
// la sesión desde la que actúa y sus roles vigentes.
type Principal struct {
	UserID    string
	SessionID string
	Roles     []domain.Role
}

// HasRole indica si la identidad tiene el rol.
func (p Principal) HasRole(role domain.Role) bool {
	return slices.Contains(p.Roles, role)
}

// HasAnyRole indica si la identidad tiene al menos uno de los roles.
func (p Principal) HasAnyRole(roles ...domain.Role) bool {
	return slices.ContainsFunc(roles, p.HasRole)
}

// CanViewWorkspace indica si la identidad puede consultar el curso o proyecto:
// el administrador ve todos y el profesor solo los suyos (RF-12.1, RN-03, RN-05).
func (p Principal) CanViewWorkspace(w domain.Workspace) bool {
	return p.HasRole(domain.RoleAdmin) || p.CanManageWorkspace(w)
}

// CanManageWorkspace indica si la identidad es el profesor responsable del
// curso o proyecto y, por tanto, puede operar sobre él (cerrarlo, vincular).
func (p Principal) CanManageWorkspace(w domain.Workspace) bool {
	return p.HasRole(domain.RoleProfessor) && w.ProfessorID == p.UserID
}
