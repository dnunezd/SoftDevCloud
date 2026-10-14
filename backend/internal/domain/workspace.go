package domain

import (
	"strings"
	"time"
)

// WorkspaceType es el tipo de espacio de trabajo (RN-06).
type WorkspaceType string

const (
	WorkspaceCourse  WorkspaceType = "CURSO"
	WorkspaceProject WorkspaceType = "PROYECTO"
)

// WorkspaceStatus es el estado de un espacio (RN-08).
type WorkspaceStatus string

const (
	WorkspaceActive WorkspaceStatus = "ACTIVO"
	WorkspaceClosed WorkspaceStatus = "CERRADO"
)

// Workspace es un espacio de trabajo: un curso o un proyecto creado por un
// profesor dentro de un período académico.
//
// Un espacio cerrado es de solo lectura y no admite registros operativos
// (RF-03.5 a RF-03.7, RN-09). El profesor responsable no se copia a las
// vinculaciones: se obtiene siempre de ProfessorID (RN-11).
type Workspace struct {
	ID           string
	PeriodID     string
	ProfessorID  string
	Type         WorkspaceType
	Name         string
	StartDate    time.Time
	EndDate      time.Time
	Observations string
	Status       WorkspaceStatus
	// ClosedAt solo tiene valor cuando Status es CERRADO.
	ClosedAt *time.Time
}

// Validate comprueba los datos mínimos del espacio (RF-03.3).
func (w Workspace) Validate() error {
	fields := map[string]string{}
	if w.Type != WorkspaceCourse && w.Type != WorkspaceProject {
		fields["type"] = "debe ser CURSO o PROYECTO"
	}
	if strings.TrimSpace(w.Name) == "" {
		fields["name"] = "es obligatorio"
	}
	if calendarDay(w.EndDate).Before(calendarDay(w.StartDate)) {
		fields["end_date"] = "no puede ser anterior a la fecha de inicio"
	}
	if len(fields) > 0 {
		return NewValidationError(fields)
	}
	return nil
}

// IsWritable indica si el espacio admite registros operativos (RN-09).
func (w Workspace) IsWritable() bool {
	return w.Status == WorkspaceActive
}

// Close cierra el espacio y registra cuándo (RF-03.5). Cerrar un espacio ya
// cerrado es un error: el cierre no se repite ni se reabre.
func (w *Workspace) Close(now time.Time) error {
	if !w.IsWritable() {
		return ErrWorkspaceClosed
	}
	w.Status = WorkspaceClosed
	w.ClosedAt = &now
	return nil
}
