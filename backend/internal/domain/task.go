package domain

import (
	"strings"
	"time"
)

// TaskStatus es el estado de una tarea. Es un catálogo cerrado (RN-27, RN-28).
type TaskStatus string

const (
	TaskOpen       TaskStatus = "ABIERTO"
	TaskInProgress TaskStatus = "EN_DESARROLLO"
	TaskDone       TaskStatus = "FINALIZADO"
)

// Valid indica si el estado pertenece al catálogo.
func (s TaskStatus) Valid() bool {
	return s == TaskOpen || s == TaskInProgress || s == TaskDone
}

// Task es una tarea semanal registrada por un monitor o asistente graduado.
//
// Pertenece a una única vinculación (RN-21) y su semana nunca cambia: si la
// actividad continúa en otra semana se crea una tarea nueva (RN-26). IsLate se
// fija al crearla y no se recalcula (RN-32).
type Task struct {
	ID           string
	AssignmentID string
	WeekStart    Week
	Title        string
	Description  string
	Status       TaskStatus
	HoursSpent   int
	Observations string
	IsLate       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// TaskInput son los datos editables de una tarea (RN-22). El tiempo se
// registra en horas enteras (RN-24).
type TaskInput struct {
	Title        string
	Description  string
	Status       string
	HoursSpent   int
	Observations string
}

// validate normaliza y valida los datos editables.
func (in TaskInput) validate() (TaskInput, TaskStatus, error) {
	fields := map[string]string{}
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		fields["title"] = "es obligatorio"
	}
	status := TaskStatus(strings.ToUpper(strings.TrimSpace(in.Status)))
	if !status.Valid() {
		fields["status"] = "debe ser ABIERTO, EN_DESARROLLO o FINALIZADO"
	}
	if in.HoursSpent < 0 {
		fields["hours_spent"] = "no puede ser negativo"
	}
	if len(fields) > 0 {
		return TaskInput{}, "", NewValidationError(fields)
	}
	return in, status, nil
}

// NewTask crea una tarea para la semana indicada dentro del espacio de la
// vinculación. Rechaza el espacio cerrado (RN-09) y las semanas futuras. Si la
// semana ya cerró, la tarea nace marcada como reporte tardío y queda inmutable
// (RN-31 a RN-33).
func NewTask(assignmentID string, week Week, in TaskInput, workspace Workspace, now time.Time) (Task, error) {
	if !workspace.IsWritable() {
		return Task{}, ErrWorkspaceClosed
	}
	if week.IsFuture(now) {
		return Task{}, NewValidationError(map[string]string{"week_start": "no puede ser una semana futura"})
	}
	data, status, err := in.validate()
	if err != nil {
		return Task{}, err
	}
	return Task{
		AssignmentID: assignmentID,
		WeekStart:    week,
		Title:        data.Title,
		Description:  data.Description,
		Status:       status,
		HoursSpent:   data.HoursSpent,
		Observations: data.Observations,
		IsLate:       week.IsClosed(now),
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// CanModify indica si la tarea se puede editar, cambiar de estado o eliminar:
// solo durante su semana activa, nunca si es un reporte tardío y nunca en un
// espacio cerrado (RN-09, RN-30, RN-33, RN-34).
func (t Task) CanModify(now time.Time, workspace Workspace) bool {
	return workspace.IsWritable() && !t.IsLate && t.WeekStart.IsActive(now)
}

// Update devuelve la tarea con los datos nuevos. La semana, la vinculación y
// la marca de tardío no cambian. Falla con ErrWorkspaceClosed si el espacio
// está cerrado y con ErrTaskNotModifiable si la tarea ya no se puede modificar.
func (t Task) Update(in TaskInput, now time.Time, workspace Workspace) (Task, error) {
	if !workspace.IsWritable() {
		return Task{}, ErrWorkspaceClosed
	}
	if !t.CanModify(now, workspace) {
		return Task{}, ErrTaskNotModifiable
	}
	data, status, err := in.validate()
	if err != nil {
		return Task{}, err
	}
	t.Title = data.Title
	t.Description = data.Description
	t.Status = status
	t.HoursSpent = data.HoursSpent
	t.Observations = data.Observations
	t.UpdatedAt = now
	return t, nil
}
