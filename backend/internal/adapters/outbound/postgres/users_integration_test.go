//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/outbound/postgres"
	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
	"github.com/dnunezd/SoftDevCloud/backend/internal/ports"
)

// Ejecutar con: TEST_DATABASE_URL=... go test -tags=integration ./internal/adapters/outbound/postgres/...
// Usa una base de datos dedicada: cada prueba la vacía antes de empezar.
func setupDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL no está definida")
	}
	ctx := context.Background()
	require.NoError(t, postgres.Migrate(url))
	pool, err := postgres.NewPool(ctx, url)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	_, err = pool.Exec(ctx, "TRUNCATE users, academic_periods CASCADE")
	require.NoError(t, err)
	return pool
}

func TestUserRepository(t *testing.T) {
	repo := postgres.NewUserRepository(setupDB(t))
	ctx := context.Background()

	ana, err := repo.Create(ctx, domain.User{
		FullName: "Ana Pérez", Email: "ana@uni.edu", PasswordHash: "hash-a",
		Roles: []domain.Role{domain.RoleGraduateAssistant, domain.RoleMonitor},
	})
	require.NoError(t, err)
	assert.Len(t, ana.ID, 36)
	assert.False(t, ana.CreatedAt.IsZero())

	_, err = repo.Create(ctx, domain.User{
		FullName: "Beto Gómez", Email: "beto@uni.edu", PasswordHash: "hash-b",
		Roles: []domain.Role{domain.RoleProfessor},
	})
	require.NoError(t, err)

	t.Run("el correo es único", func(t *testing.T) {
		_, err := repo.Create(ctx, domain.User{
			FullName: "Otra Ana", Email: "ana@uni.edu", PasswordHash: "h", Roles: []domain.Role{domain.RoleMonitor},
		})
		assert.ErrorIs(t, err, domain.ErrEmailAlreadyExists)
	})

	t.Run("consulta por correo y por ID con sus roles", func(t *testing.T) {
		byEmail, err := repo.GetByEmail(ctx, "ana@uni.edu")
		require.NoError(t, err)
		assert.Equal(t, ana.ID, byEmail.ID)
		assert.Equal(t, "hash-a", byEmail.PasswordHash)
		assert.Equal(t, []domain.Role{domain.RoleGraduateAssistant, domain.RoleMonitor}, byEmail.Roles)

		byID, err := repo.GetByID(ctx, ana.ID)
		require.NoError(t, err)
		assert.Equal(t, byEmail, byID)
	})

	t.Run("usuario inexistente o ID mal formado", func(t *testing.T) {
		_, err := repo.GetByID(ctx, "00000000-0000-0000-0000-000000000000")
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		_, err = repo.GetByID(ctx, "no-es-un-uuid")
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		_, err = repo.GetByEmail(ctx, "nadie@uni.edu")
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("lista con filtro por rol y paginación", func(t *testing.T) {
		all, err := repo.List(ctx, ports.UserFilter{Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, 2, all.Total)
		require.Len(t, all.Users, 2)
		assert.Equal(t, "Ana Pérez", all.Users[0].FullName, "ordenado por nombre")

		monitors, err := repo.List(ctx, ports.UserFilter{Role: domain.RoleMonitor, Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, 1, monitors.Total)
		require.Len(t, monitors.Users, 1)
		assert.Len(t, monitors.Users[0].Roles, 2, "el filtro no recorta los demás roles del usuario")

		second, err := repo.List(ctx, ports.UserFilter{Limit: 1, Offset: 1})
		require.NoError(t, err)
		assert.Equal(t, 2, second.Total)
		require.Len(t, second.Users, 1)
		assert.Equal(t, "Beto Gómez", second.Users[0].FullName)
	})

	t.Run("reemplaza roles y cuenta por rol", func(t *testing.T) {
		require.NoError(t, repo.ReplaceRoles(ctx, ana.ID, []domain.Role{domain.RoleAdmin}))

		updated, err := repo.GetByID(ctx, ana.ID)
		require.NoError(t, err)
		assert.Equal(t, []domain.Role{domain.RoleAdmin}, updated.Roles)

		admins, err := repo.CountWithRole(ctx, domain.RoleAdmin)
		require.NoError(t, err)
		assert.Equal(t, 1, admins)

		err = repo.ReplaceRoles(ctx, "00000000-0000-0000-0000-000000000000", []domain.Role{domain.RoleAdmin})
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}
