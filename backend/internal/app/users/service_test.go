package users_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dnunezd/SoftDevCloud/backend/internal/app/users"
	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
	"github.com/dnunezd/SoftDevCloud/backend/internal/mocks"
	"github.com/dnunezd/SoftDevCloud/backend/internal/ports"
)

const userID = "8c1f4f0e-3f5b-4d36-9a52-1d2f1f0e8a11"

type fixture struct {
	service *users.Service
	repo    *mocks.UserRepository
	hasher  *mocks.PasswordHasher
}

func newFixture(t *testing.T) fixture {
	f := fixture{repo: mocks.NewUserRepository(t), hasher: mocks.NewPasswordHasher(t)}
	f.service = users.NewService(f.repo, f.hasher)
	return f
}

func requireValidationFields(t *testing.T, err error, fields ...string) {
	t.Helper()
	var derr *domain.Error
	require.ErrorAs(t, err, &derr)
	require.Equal(t, domain.KindValidation, derr.Kind)
	for _, field := range fields {
		assert.Contains(t, derr.Fields, field)
	}
}

func TestCreate_StoresNormalizedUserWithHashedPassword(t *testing.T) {
	f := newFixture(t)
	toStore := domain.User{
		FullName:     "Ana Pérez",
		Email:        "ana@uni.edu",
		PasswordHash: "hash-bcrypt",
		Roles:        []domain.Role{domain.RoleMonitor, domain.RoleProfessor},
	}
	stored := toStore
	stored.ID = userID
	stored.CreatedAt = time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	f.hasher.EXPECT().Hash("secreta123").Return("hash-bcrypt", nil)
	f.repo.EXPECT().Create(mock.Anything, toStore).Return(stored, nil)

	user, err := f.service.Create(context.Background(), users.CreateInput{
		FullName: " Ana Pérez ",
		Email:    "ANA@uni.edu",
		Password: "secreta123",
		Roles:    []string{"PROFESOR", "monitor"},
	})

	require.NoError(t, err)
	assert.Equal(t, stored, user)
}

func TestCreate_InvalidDataNeverReachesTheRepository(t *testing.T) {
	f := newFixture(t) // sin expectativas: cualquier llamada al repositorio o al hasher falla

	_, err := f.service.Create(context.Background(), users.CreateInput{Email: "no-es-correo", Password: "corta"})

	requireValidationFields(t, err, "full_name", "email", "password", "roles")
}

func TestCreate_DuplicatedEmailIsAConflict(t *testing.T) {
	f := newFixture(t)
	f.hasher.EXPECT().Hash("secreta123").Return("hash-bcrypt", nil)
	f.repo.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.User{}, domain.ErrEmailAlreadyExists)

	_, err := f.service.Create(context.Background(), users.CreateInput{
		FullName: "Ana", Email: "ana@uni.edu", Password: "secreta123", Roles: []string{"MONITOR"},
	})

	assert.ErrorIs(t, err, domain.ErrEmailAlreadyExists)
}

func TestList(t *testing.T) {
	t.Run("aplica la paginación por defecto", func(t *testing.T) {
		f := newFixture(t)
		page := ports.UserPage{Users: []domain.User{{ID: userID}}, Total: 1}
		f.repo.EXPECT().List(mock.Anything, ports.UserFilter{Limit: users.DefaultPageSize}).Return(page, nil)

		result, err := f.service.List(context.Background(), users.ListInput{})

		require.NoError(t, err)
		assert.Equal(t, users.Page{Users: page.Users, Total: 1, Limit: users.DefaultPageSize, Offset: 0}, result)
	})

	t.Run("filtra por rol sin distinguir mayúsculas", func(t *testing.T) {
		f := newFixture(t)
		filter := ports.UserFilter{Role: domain.RoleMonitor, Limit: 10, Offset: 20}
		f.repo.EXPECT().List(mock.Anything, filter).Return(ports.UserPage{}, nil)

		_, err := f.service.List(context.Background(), users.ListInput{Role: "monitor", Limit: 10, Offset: 20})

		require.NoError(t, err)
	})

	t.Run("rechaza parámetros inválidos", func(t *testing.T) {
		f := newFixture(t)

		_, err := f.service.List(context.Background(), users.ListInput{Role: "DECANO", Limit: users.MaxPageSize + 1, Offset: -1})

		requireValidationFields(t, err, "role", "limit", "offset")
	})
}

func TestSetRoles(t *testing.T) {
	monitor := domain.User{ID: userID, Email: "ana@uni.edu", Roles: []domain.Role{domain.RoleMonitor}}
	admin := domain.User{ID: userID, Email: "admin@uni.edu", Roles: []domain.Role{domain.RoleAdmin}}

	t.Run("reemplaza el conjunto de roles", func(t *testing.T) {
		f := newFixture(t)
		newRoles := []domain.Role{domain.RoleGraduateAssistant, domain.RoleMonitor}
		f.repo.EXPECT().GetByID(mock.Anything, userID).Return(monitor, nil)
		f.repo.EXPECT().ReplaceRoles(mock.Anything, userID, newRoles).Return(nil)

		user, err := f.service.SetRoles(context.Background(), userID, []string{"MONITOR", "asistente_graduado"})

		require.NoError(t, err)
		assert.Equal(t, newRoles, user.Roles)
	})

	t.Run("impide quitar el rol ADMIN al último administrador", func(t *testing.T) {
		f := newFixture(t)
		f.repo.EXPECT().GetByID(mock.Anything, userID).Return(admin, nil)
		f.repo.EXPECT().CountWithRole(mock.Anything, domain.RoleAdmin).Return(1, nil)

		_, err := f.service.SetRoles(context.Background(), userID, []string{"PROFESOR"})

		assert.ErrorIs(t, err, domain.ErrLastAdmin)
	})

	t.Run("permite quitar el rol ADMIN si hay otros administradores", func(t *testing.T) {
		f := newFixture(t)
		f.repo.EXPECT().GetByID(mock.Anything, userID).Return(admin, nil)
		f.repo.EXPECT().CountWithRole(mock.Anything, domain.RoleAdmin).Return(2, nil)
		f.repo.EXPECT().ReplaceRoles(mock.Anything, userID, []domain.Role{domain.RoleProfessor}).Return(nil)

		_, err := f.service.SetRoles(context.Background(), userID, []string{"PROFESOR"})

		require.NoError(t, err)
	})

	t.Run("usuario inexistente", func(t *testing.T) {
		f := newFixture(t)
		f.repo.EXPECT().GetByID(mock.Anything, userID).Return(domain.User{}, domain.ErrUserNotFound)

		_, err := f.service.SetRoles(context.Background(), userID, []string{"MONITOR"})

		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("roles inválidos no consultan el repositorio", func(t *testing.T) {
		f := newFixture(t)

		_, err := f.service.SetRoles(context.Background(), userID, nil)

		requireValidationFields(t, err, "roles")
	})
}

func TestEnsureAdmin(t *testing.T) {
	t.Run("no crea nada si ya existe un administrador", func(t *testing.T) {
		f := newFixture(t)
		f.repo.EXPECT().CountWithRole(mock.Anything, domain.RoleAdmin).Return(1, nil)

		created, err := f.service.EnsureAdmin(context.Background(), "Admin", "admin@uni.edu", "secreta123")

		require.NoError(t, err)
		assert.False(t, created)
	})

	t.Run("crea el administrador inicial", func(t *testing.T) {
		f := newFixture(t)
		f.repo.EXPECT().CountWithRole(mock.Anything, domain.RoleAdmin).Return(0, nil)
		f.hasher.EXPECT().Hash("secreta123").Return("hash-bcrypt", nil)
		f.repo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(u domain.User) bool {
			return u.Email == "admin@uni.edu" && u.HasRole(domain.RoleAdmin) && len(u.Roles) == 1
		})).Return(domain.User{ID: userID}, nil)

		created, err := f.service.EnsureAdmin(context.Background(), "Admin", "admin@uni.edu", "secreta123")

		require.NoError(t, err)
		assert.True(t, created)
	})

	t.Run("tolera que otra instancia lo cree en paralelo", func(t *testing.T) {
		f := newFixture(t)
		f.repo.EXPECT().CountWithRole(mock.Anything, domain.RoleAdmin).Return(0, nil)
		f.hasher.EXPECT().Hash("secreta123").Return("hash-bcrypt", nil)
		f.repo.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.User{}, domain.ErrEmailAlreadyExists)

		created, err := f.service.EnsureAdmin(context.Background(), "Admin", "admin@uni.edu", "secreta123")

		require.NoError(t, err)
		assert.False(t, created)
	})
}
