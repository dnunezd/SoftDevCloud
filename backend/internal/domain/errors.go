package domain

// ErrorKind clasifica los errores de dominio para que los adaptadores de entrada
// los traduzcan (por ejemplo, a códigos HTTP) sin conocer cada error concreto.
type ErrorKind int

const (
	KindValidation ErrorKind = iota + 1
	KindNotFound
	KindConflict
	KindUnauthorized
	KindForbidden
)

// Error es un error de dominio con un código estable para los clientes de la API.
type Error struct {
	Kind    ErrorKind
	Code    string
	Message string
	// Fields detalla, para errores de validación, el problema de cada campo.
	Fields map[string]string
}

func (e *Error) Error() string {
	return e.Code + ": " + e.Message
}

// Errores de dominio conocidos. Se comparan con errors.Is.
var (
	ErrInvalidCredentials = &Error{Kind: KindUnauthorized, Code: "invalid_credentials", Message: "credenciales inválidas"}
	ErrUnauthenticated    = &Error{Kind: KindUnauthorized, Code: "unauthenticated", Message: "se requiere autenticación"}
	ErrSessionRevoked     = &Error{Kind: KindUnauthorized, Code: "session_revoked", Message: "la sesión fue cerrada o venció; inicie sesión de nuevo"}
	ErrSessionNotFound    = &Error{Kind: KindNotFound, Code: "session_not_found", Message: "la sesión no existe o no está activa"}
	ErrForbidden          = &Error{Kind: KindForbidden, Code: "forbidden", Message: "no tiene permisos para esta operación"}
	ErrUserNotFound       = &Error{Kind: KindNotFound, Code: "user_not_found", Message: "el usuario no existe"}
	ErrEmailAlreadyExists = &Error{Kind: KindConflict, Code: "email_already_exists", Message: "ya existe un usuario con ese correo"}
	ErrLastAdmin          = &Error{Kind: KindConflict, Code: "last_admin", Message: "no se puede quitar el rol ADMIN al último administrador"}
)

// Errores de períodos y espacios.
var (
	ErrPeriodNotFound    = &Error{Kind: KindNotFound, Code: "period_not_found", Message: "el período académico no existe"}
	ErrPeriodCodeExists  = &Error{Kind: KindConflict, Code: "period_code_already_exists", Message: "ya existe un período con ese código"}
	ErrPeriodClosed      = &Error{Kind: KindConflict, Code: "period_closed", Message: "el período académico ya está cerrado"}
	ErrWorkspaceNotFound = &Error{Kind: KindNotFound, Code: "workspace_not_found", Message: "el curso o proyecto no existe"}
)

// Errores de espacios, tareas y reportes.
var (
	ErrCreationWindowClosed = &Error{Kind: KindConflict, Code: "creation_window_closed", Message: "el período no admite crear cursos ni proyectos: está cerrado o venció su ventana de creación"}
	ErrWorkspaceClosed      = &Error{Kind: KindConflict, Code: "workspace_closed", Message: "el curso o proyecto está cerrado y solo admite consulta"}
	ErrTaskNotModifiable    = &Error{Kind: KindConflict, Code: "task_not_modifiable", Message: "la tarea solo se modifica o elimina durante su semana activa y no si es un reporte tardío"}
	ErrInvalidReportState   = &Error{Kind: KindConflict, Code: "invalid_report_transition", Message: "transición de estado no permitida para el reporte"}
)

// Errores de las reglas de carga horaria (RN-14 a RN-19). Son validaciones
// bloqueantes (RF-05.6): la API responde 422 e indica cuál regla se incumplió.
var (
	ErrGraduateHoursExceeded = &Error{Kind: KindValidation, Code: "graduate_hours_exceeded", Message: "la persona superaría las 22 horas semanales como asistente graduado"}
	ErrMonitorCountExceeded  = &Error{Kind: KindValidation, Code: "monitor_count_exceeded", Message: "la persona superaría las 3 monitorías simultáneas"}
	ErrMonitorHoursExceeded  = &Error{Kind: KindValidation, Code: "monitor_hours_exceeded", Message: "la persona superaría las 12 horas semanales como monitor"}
	ErrMixedHoursExceeded    = &Error{Kind: KindValidation, Code: "mixed_hours_exceeded", Message: "las horas de monitoría superarían el 40 % de las horas como asistente graduado"}
)

// NewValidationError crea un error de validación con el detalle por campo.
func NewValidationError(fields map[string]string) *Error {
	return &Error{
		Kind:    KindValidation,
		Code:    "validation_failed",
		Message: "los datos enviados no son válidos",
		Fields:  fields,
	}
}
