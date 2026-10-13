//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/outbound/postgres"
	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
)

func TestSessionRepository(t *testing.T) {
	pool := setupDB(t)
	users := postgres.NewUserRepository(pool)
	sessions := postgres.NewSessionRepository(pool)
	ctx := context.Background()

	ana, err := users.Create(ctx, domain.User{
		FullName: "Ana Pérez", Email: "ana@uni.edu", PasswordHash: "h",
		Roles: []domain.Role{domain.RoleGraduateAssistant, domain.RoleMonitor},
	})
	require.NoError(t, err)
	beto, err := users.Create(ctx, domain.User{
		FullName: "Beto Gómez", Email: "beto@uni.edu", PasswordHash: "h", Roles: []domain.Role{domain.RoleProfessor},
	})
	require.NoError(t, err)

	open := func(userID, agent string, ttl time.Duration) domain.Session {
		t.Helper()
		s, err := sessions.Create(ctx, domain.NewSession(userID, agent, "10.0.0.7", time.Now(), ttl))
		require.NoError(t, err)
		return s
	}
	laptop := open(ana.ID, "portátil", time.Hour)
	phone := open(ana.ID, "celular", time.Hour)

	t.Run("una sesión activa devuelve los roles vigentes del usuario", func(t *testing.T) {
		assert.Len(t, laptop.ID, 36)
		roles, err := sessions.ActiveRoles(ctx, laptop.ID, ana.ID)
		require.NoError(t, err)
		assert.Equal(t, []domain.Role{domain.RoleGraduateAssistant, domain.RoleMonitor}, roles)

		require.NoError(t, users.ReplaceRoles(ctx, ana.ID, []domain.Role{domain.RoleMonitor}))
		roles, err = sessions.ActiveRoles(ctx, laptop.ID, ana.ID)
		require.NoError(t, err)
		assert.Equal(t, []domain.Role{domain.RoleMonitor}, roles, "un cambio de roles se refleja de inmediato")
	})

	t.Run("una sesión no sirve con otro usuario ni con un ID mal formado", func(t *testing.T) {
		_, err := sessions.ActiveRoles(ctx, laptop.ID, beto.ID)
		assert.ErrorIs(t, err, domain.ErrSessionNotFound)
		_, err = sessions.ActiveRoles(ctx, "no-es-un-uuid", ana.ID)
		assert.ErrorIs(t, err, domain.ErrSessionNotFound)
	})

	t.Run("una sesión vencida no está activa", func(t *testing.T) {
		expired := open(beto.ID, "viejo", -time.Minute)

		_, err := sessions.ActiveRoles(ctx, expired.ID, beto.ID)
		assert.ErrorIs(t, err, domain.ErrSessionNotFound)

		active, err := sessions.ListActive(ctx, beto.ID)
		require.NoError(t, err)
		assert.Empty(t, active)

		purged, err := sessions.DeleteExpired(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, purged)
	})

	t.Run("cerrar una sesión no afecta las demás del usuario", func(t *testing.T) {
		require.NoError(t, sessions.Revoke(ctx, phone.ID))
		require.NoError(t, sessions.Revoke(ctx, phone.ID), "cerrar dos veces no es un error")

		_, err := sessions.ActiveRoles(ctx, phone.ID, ana.ID)
		assert.ErrorIs(t, err, domain.ErrSessionNotFound)
		_, err = sessions.ActiveRoles(ctx, laptop.ID, ana.ID)
		assert.NoError(t, err)

		active, err := sessions.ListActive(ctx, ana.ID)
		require.NoError(t, err)
		require.Len(t, active, 1)
		assert.Equal(t, "portátil", active[0].UserAgent)
	})

	t.Run("cerrar todas las sesiones de un usuario", func(t *testing.T) {
		open(ana.ID, "tableta", time.Hour)

		revoked, err := sessions.RevokeAll(ctx, ana.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, revoked)

		active, err := sessions.ListActive(ctx, ana.ID)
		require.NoError(t, err)
		assert.Empty(t, active)
	})

	t.Run("al abrir una sesión se borran las vencidas del mismo usuario", func(t *testing.T) {
		open(beto.ID, "vencida", -time.Minute)
		open(beto.ID, "nueva", time.Hour)

		var stored int
		require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM sessions WHERE user_id = $1", beto.ID).Scan(&stored))
		assert.Equal(t, 1, stored)
	})
}
