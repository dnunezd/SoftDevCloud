package domain_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
)

var validRegistration = domain.UserRegistration{
	FullName: "  Ana Pérez ",
	Email:    " Ana.Perez@Uni.EDU.co ",
	Password: "secreta123",
	Roles:    []string{"monitor", "ASISTENTE_GRADUADO", "MONITOR"},
}

func TestUserRegistrationValidate_NormalizesValidData(t *testing.T) {
	user, err := validRegistration.Validate()

	require.NoError(t, err)
	assert.Equal(t, "Ana Pérez", user.FullName)
	assert.Equal(t, "ana.perez@uni.edu.co", user.Email)
	assert.Equal(t, []domain.Role{domain.RoleGraduateAssistant, domain.RoleMonitor}, user.Roles,
		"los roles se deduplican y ordenan")
	assert.Empty(t, user.PasswordHash, "el dominio nunca calcula el hash")
}

func TestUserRegistrationValidate_RejectsInvalidFields(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*domain.UserRegistration)
		field   string
		message string
	}{
		{"nombre vacío", func(r *domain.UserRegistration) { r.FullName = "   " }, "full_name", "es obligatorio"},
		{"nombre demasiado largo", func(r *domain.UserRegistration) { r.FullName = strings.Repeat("a", 151) }, "full_name", "no puede superar 150 caracteres"},
		{"correo vacío", func(r *domain.UserRegistration) { r.Email = " " }, "email", "es obligatorio"},
		{"correo sin arroba", func(r *domain.UserRegistration) { r.Email = "ana-uni.edu" }, "email", "no tiene un formato válido"},
		{"correo con nombre visible", func(r *domain.UserRegistration) { r.Email = "Ana <ana@uni.edu>" }, "email", "no tiene un formato válido"},
		{"contraseña corta", func(r *domain.UserRegistration) { r.Password = "1234567" }, "password", "debe tener al menos 8 caracteres"},
		{"contraseña de más de 72 bytes", func(r *domain.UserRegistration) { r.Password = strings.Repeat("ñ", 37) }, "password", "no puede superar 72 bytes"},
		{"sin roles", func(r *domain.UserRegistration) { r.Roles = nil }, "roles", "debe indicar al menos un rol"},
		{"rol desconocido", func(r *domain.UserRegistration) { r.Roles = []string{"DECANO"} }, "roles", "rol desconocido: DECANO"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reg := validRegistration
			reg.Roles = slices.Clone(validRegistration.Roles)
			tc.mutate(&reg)

			_, err := reg.Validate()

			var derr *domain.Error
			require.ErrorAs(t, err, &derr)
			assert.Equal(t, domain.KindValidation, derr.Kind)
			assert.Equal(t, map[string]string{tc.field: tc.message}, derr.Fields)
		})
	}
}

func TestUserRegistrationValidate_ReportsAllFieldsAtOnce(t *testing.T) {
	_, err := domain.UserRegistration{}.Validate()

	var derr *domain.Error
	require.ErrorAs(t, err, &derr)
	assert.Len(t, derr.Fields, 4)
	for _, field := range []string{"full_name", "email", "password", "roles"} {
		assert.Contains(t, derr.Fields, field)
	}
}

func TestParseRoles(t *testing.T) {
	t.Run("acepta varios roles sin distinguir mayúsculas", func(t *testing.T) {
		roles, err := domain.ParseRoles([]string{" profesor", "ADMIN", "Profesor"})

		require.NoError(t, err)
		assert.Equal(t, []domain.Role{domain.RoleAdmin, domain.RoleProfessor}, roles)
	})

	for name, input := range map[string][]string{
		"lista vacía":     {},
		"rol desconocido": {"MONITOR", "COORDINADOR"},
	} {
		t.Run("rechaza "+name, func(t *testing.T) {
			_, err := domain.ParseRoles(input)

			var derr *domain.Error
			require.ErrorAs(t, err, &derr)
			assert.Equal(t, domain.KindValidation, derr.Kind)
			assert.Contains(t, derr.Fields, "roles")
		})
	}
}

func TestUserHasRole(t *testing.T) {
	user := domain.User{Roles: []domain.Role{domain.RoleMonitor, domain.RoleGraduateAssistant}}

	assert.True(t, user.HasRole(domain.RoleMonitor))
	assert.True(t, user.HasRole(domain.RoleGraduateAssistant))
	assert.False(t, user.HasRole(domain.RoleAdmin))
}

func TestNormalizeEmail(t *testing.T) {
	assert.Equal(t, "ana@uni.edu", domain.NormalizeEmail("  ANA@Uni.Edu "))
}
