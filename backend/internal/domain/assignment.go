package domain

import "time"

// Assignment es la vinculación de una persona a un curso o proyecto con un rol
// (MONITOR o ASISTENTE_GRADUADO) y unas horas semanales contratadas (RN-02,
// RN-10, RN-11, RF-04). Es el punto del que cuelgan las tareas.
//
// Las horas contratadas son inmutables (RN-12): no hay operación de edición
// para el profesor ni para el usuario operativo; solo el administrador puede
// rectificarlas con Rectify, y queda registro del antes y el después (RF-04.5).
type Assignment struct {
	ID          string
	UserID      string
	WorkspaceID string
	Role        Role
	WeeklyHours int
	CreatedAt   time.Time
}

// Validate comprueba el rol y las horas de la vinculación. Cada vinculación
// individual tampoco puede superar el tope de su rol (RN-14, RN-16).
func (a Assignment) Validate() error {
	fields := map[string]string{}
	maxHours := 0
	switch a.Role {
	case RoleMonitor:
		maxHours = MaxMonitorHours
	case RoleGraduateAssistant:
		maxHours = MaxGraduateAssistantHours
	default:
		fields["role"] = "debe ser MONITOR o ASISTENTE_GRADUADO"
	}
	switch {
	case a.WeeklyHours <= 0:
		fields["weekly_hours"] = "debe ser mayor que cero"
	case maxHours > 0 && a.WeeklyHours > maxHours:
		fields["weekly_hours"] = "supera el máximo semanal del rol"
	}
	if len(fields) > 0 {
		return NewValidationError(fields)
	}
	return nil
}

// IsActive indica si la vinculación cuenta para los límites de horas: está
// activa mientras su espacio lo esté (RN-13).
func (a Assignment) IsActive(workspace Workspace) bool {
	return workspace.Status == WorkspaceActive
}

// AssignmentChanges son los campos que el administrador puede rectificar. Un
// campo nulo no se modifica.
type AssignmentChanges struct {
	WeeklyHours *int
	Role        *Role
}

// AssignmentAudit registra una rectificación del administrador con el estado
// de la vinculación antes y después (RF-04.5).
type AssignmentAudit struct {
	ID           string
	AssignmentID string
	AdminID      string
	Before       Assignment
	After        Assignment
	ChangedAt    time.Time
}

// Rectify aplica una rectificación administrativa y devuelve la vinculación
// corregida junto con su registro de auditoría. No evalúa la carga horaria:
// quien la invoca debe revalidarla con WorkloadRule, excluyendo esta
// vinculación del conjunto de activas.
func (a Assignment) Rectify(changes AssignmentChanges, adminID string, now time.Time) (Assignment, AssignmentAudit, error) {
	if changes.WeeklyHours == nil && changes.Role == nil {
		return Assignment{}, AssignmentAudit{}, NewValidationError(map[string]string{"changes": "debe indicar al menos un campo a rectificar"})
	}
	updated := a
	if changes.WeeklyHours != nil {
		updated.WeeklyHours = *changes.WeeklyHours
	}
	if changes.Role != nil {
		updated.Role = *changes.Role
	}
	if err := updated.Validate(); err != nil {
		return Assignment{}, AssignmentAudit{}, err
	}
	audit := AssignmentAudit{AssignmentID: a.ID, AdminID: adminID, Before: a, After: updated, ChangedAt: now}
	return updated, audit, nil
}
