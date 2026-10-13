package domain

import (
	"net/mail"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// Role es un rol global del sistema. Un usuario puede tener varios (RN-01, RF-01.3).
type Role string

const (
	RoleAdmin             Role = "ADMIN"
	RoleProfessor         Role = "PROFESOR"
	RoleMonitor           Role = "MONITOR"
	RoleGraduateAssistant Role = "ASISTENTE_GRADUADO"
)

// allRoles es el catálogo cerrado de roles.
var allRoles = []Role{RoleAdmin, RoleProfessor, RoleMonitor, RoleGraduateAssistant}

// Valid indica si el rol pertenece al catálogo.
func (r Role) Valid() bool {
	return slices.Contains(allRoles, r)
}

// Políticas de los datos de usuario.
const (
	MaxFullNameLength = 150
	MaxEmailLength    = 254
	MinPasswordLength = 8
	// MaxPasswordBytes es el límite de bcrypt: los bytes adicionales se ignorarían.
	MaxPasswordBytes = 72
)

// User es una persona con credenciales locales y uno o más roles globales.
type User struct {
	ID           string
	FullName     string
	Email        string
	PasswordHash string
	Roles        []Role
	CreatedAt    time.Time
}

// HasRole indica si el usuario tiene el rol.
func (u User) HasRole(role Role) bool {
	return slices.Contains(u.Roles, role)
}

// UserRegistration son los datos con los que el administrador crea un usuario (RF-02.1).
type UserRegistration struct {
	FullName string
	Email    string
	Password string
	Roles    []string
}

// Validate normaliza y valida el registro. Devuelve el usuario (sin hash de
// contraseña) o un error de validación con el detalle de todos los campos inválidos.
func (r UserRegistration) Validate() (User, error) {
	fields := map[string]string{}

	fullName := strings.TrimSpace(r.FullName)
	switch {
	case fullName == "":
		fields["full_name"] = "es obligatorio"
	case utf8.RuneCountInString(fullName) > MaxFullNameLength:
		fields["full_name"] = "no puede superar 150 caracteres"
	}

	email, msg := validateEmail(r.Email)
	if msg != "" {
		fields["email"] = msg
	}

	if msg := validatePassword(r.Password); msg != "" {
		fields["password"] = msg
	}

	roles, msg := parseRoles(r.Roles)
	if msg != "" {
		fields["roles"] = msg
	}

	if len(fields) > 0 {
		return User{}, NewValidationError(fields)
	}
	return User{FullName: fullName, Email: email, Roles: roles}, nil
}

// ParseRoles valida y normaliza una lista de roles: al menos uno, solo del
// catálogo, sin duplicados y en orden estable (RF-02.2). No distingue mayúsculas.
func ParseRoles(values []string) ([]Role, error) {
	roles, msg := parseRoles(values)
	if msg != "" {
		return nil, NewValidationError(map[string]string{"roles": msg})
	}
	return roles, nil
}

// NormalizeEmail aplica la forma canónica con la que se guardan los correos.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func parseRoles(values []string) ([]Role, string) {
	if len(values) == 0 {
		return nil, "debe indicar al menos un rol"
	}
	roles := make([]Role, 0, len(values))
	for _, value := range values {
		role := Role(strings.ToUpper(strings.TrimSpace(value)))
		if !role.Valid() {
			return nil, "rol desconocido: " + value
		}
		if !slices.Contains(roles, role) {
			roles = append(roles, role)
		}
	}
	slices.Sort(roles)
	return roles, ""
}

func validateEmail(raw string) (string, string) {
	email := NormalizeEmail(raw)
	if email == "" {
		return "", "es obligatorio"
	}
	if len(email) > MaxEmailLength {
		return "", "no puede superar 254 caracteres"
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "", "no tiene un formato válido"
	}
	return email, ""
}

func validatePassword(password string) string {
	switch {
	case utf8.RuneCountInString(password) < MinPasswordLength:
		return "debe tener al menos 8 caracteres"
	case len(password) > MaxPasswordBytes:
		return "no puede superar 72 bytes"
	default:
		return ""
	}
}
