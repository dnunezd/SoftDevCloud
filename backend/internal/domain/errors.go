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

// NewValidationError crea un error de validación con el detalle por campo.
func NewValidationError(fields map[string]string) *Error {
	return &Error{
		Kind:    KindValidation,
		Code:    "validation_failed",
		Message: "los datos enviados no son válidos",
		Fields:  fields,
	}
}
