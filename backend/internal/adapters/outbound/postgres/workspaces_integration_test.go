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
	"github.com/dnunezd/SoftDevCloud/backend/internal/ports"
)

const unknownID = "00000000-0000-0000-0000-000000000000"

func day(year int, month time.Month, d int) time.Time {
	return time.Date(year, month, d, 0, 0, 0, 0, time.UTC)
}

func TestPeriodRepository(t *testing.T) {
	repo := postgres.NewPeriodRepository(setupDB(t))
	ctx := context.Background()

	second, err := repo.Create(ctx, domain.AcademicPeriod{
		Code: "2026-20", StartDate: day(2026, time.August, 1), EndDate: day(2026, time.December, 15),
		CreationDeadline: day(2026, time.August, 31), Status: domain.PeriodActive,
	})
	require.NoError(t, err)
	assert.Len(t, second.ID, 36)

	first, err := repo.Create(ctx, domain.AcademicPeriod{
		Code: "2026-10", StartDate: day(2026, time.February, 1), EndDate: day(2026, time.June, 15),
		CreationDeadline: day(2026, time.February, 28), Status: domain.PeriodActive,
	})
	require.NoError(t, err)

	t.Run("el código es único", func(t *testing.T) {
		_, err := repo.Create(ctx, domain.AcademicPeriod{
			Code: "2026-10", StartDate: day(2027, time.February, 1), EndDate: day(2027, time.June, 15),
			CreationDeadline: day(2027, time.February, 28), Status: domain.PeriodActive,
		})
		assert.ErrorIs(t, err, domain.ErrPeriodCodeExists)
	})

	t.Run("consulta por ID conserva las fechas de calendario", func(t *testing.T) {
		got, err := repo.GetByID(ctx, first.ID)
		require.NoError(t, err)
		assert.Equal(t, first, got)
		assert.Equal(t, day(2026, time.February, 28), got.CreationDeadline)
	})

	t.Run("período inexistente o ID mal formado", func(t *testing.T) {
		_, err := repo.GetByID(ctx, unknownID)
		assert.ErrorIs(t, err, domain.ErrPeriodNotFound)
		_, err = repo.GetByID(ctx, "no-es-un-uuid")
		assert.ErrorIs(t, err, domain.ErrPeriodNotFound)
	})

	t.Run("lista del período que inicia más tarde al más antiguo", func(t *testing.T) {
		list, err := repo.List(ctx)
		require.NoError(t, err)
		require.Len(t, list, 2)
		assert.Equal(t, "2026-20", list[0].Code)
		assert.Equal(t, "2026-10", list[1].Code)
	})

	t.Run("cierra una sola vez", func(t *testing.T) {
		require.NoError(t, repo.MarkClosed(ctx, first.ID))

		got, err := repo.GetByID(ctx, first.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.PeriodClosed, got.Status)
		assert.False(t, got.AcceptsWorkspaceCreation(day(2026, time.February, 10)))

		assert.ErrorIs(t, repo.MarkClosed(ctx, first.ID), domain.ErrPeriodClosed)
		assert.ErrorIs(t, repo.MarkClosed(ctx, unknownID), domain.ErrPeriodNotFound)
		assert.ErrorIs(t, repo.MarkClosed(ctx, "no-es-un-uuid"), domain.ErrPeriodNotFound)
	})
}

func TestWorkspaceRepository(t *testing.T) {
	pool := setupDB(t)
	ctx := context.Background()
	users := postgres.NewUserRepository(pool)
	periods := postgres.NewPeriodRepository(pool)
	repo := postgres.NewWorkspaceRepository(pool)

	newProfessor := func(name, email string) domain.User {
		user, err := users.Create(ctx, domain.User{FullName: name, Email: email, PasswordHash: "h", Roles: []domain.Role{domain.RoleProfessor}})
		require.NoError(t, err)
		return user
	}
	newPeriod := func(code string, start time.Time) domain.AcademicPeriod {
		period, err := periods.Create(ctx, domain.AcademicPeriod{
			Code: code, StartDate: start, EndDate: start.AddDate(0, 4, 0), CreationDeadline: start.AddDate(0, 0, 27), Status: domain.PeriodActive,
		})
		require.NoError(t, err)
		return period
	}
	newWorkspace := func(professor domain.User, period domain.AcademicPeriod, kind domain.WorkspaceType, name string, start time.Time) domain.Workspace {
		workspace, err := repo.Create(ctx, domain.Workspace{
			PeriodID: period.ID, ProfessorID: professor.ID, Type: kind, Name: name,
			StartDate: start, EndDate: start.AddDate(0, 4, 0), Observations: "obs", Status: domain.WorkspaceActive,
		})
		require.NoError(t, err)
		return workspace
	}

	ana := newProfessor("Ana Pérez", "ana@uni.edu")
	beto := newProfessor("Beto Gómez", "beto@uni.edu")
	period1 := newPeriod("2026-10", day(2026, time.February, 1))
	period2 := newPeriod("2026-20", day(2026, time.August, 1))
	arquitectura := newWorkspace(ana, period1, domain.WorkspaceCourse, "Arquitectura de Software", day(2026, time.February, 2))
	proyecto := newWorkspace(ana, period2, domain.WorkspaceProject, "Proyecto de grado", day(2026, time.August, 3))
	bases := newWorkspace(beto, period1, domain.WorkspaceCourse, "Bases de datos", day(2026, time.February, 2))

	t.Run("crea y consulta por ID con todos sus datos", func(t *testing.T) {
		assert.Len(t, arquitectura.ID, 36)

		got, err := repo.GetByID(ctx, arquitectura.ID)
		require.NoError(t, err)
		assert.Equal(t, arquitectura, got)
		assert.Equal(t, domain.WorkspaceActive, got.Status)
		assert.Nil(t, got.ClosedAt)
		assert.Equal(t, day(2026, time.February, 2), got.StartDate)
	})

	t.Run("espacio inexistente o ID mal formado", func(t *testing.T) {
		_, err := repo.GetByID(ctx, unknownID)
		assert.ErrorIs(t, err, domain.ErrWorkspaceNotFound)
		_, err = repo.GetByID(ctx, "no-es-un-uuid")
		assert.ErrorIs(t, err, domain.ErrWorkspaceNotFound)
	})

	t.Run("lista solo los espacios del profesor, del más reciente al más antiguo", func(t *testing.T) {
		page, err := repo.List(ctx, ports.WorkspaceFilter{ProfessorID: ana.ID, Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, 2, page.Total)
		require.Len(t, page.Workspaces, 2)
		assert.Equal(t, proyecto.ID, page.Workspaces[0].ID)
		assert.Equal(t, arquitectura.ID, page.Workspaces[1].ID)
	})

	t.Run("sin filtro de profesor lista todos y pagina", func(t *testing.T) {
		all, err := repo.List(ctx, ports.WorkspaceFilter{Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, 3, all.Total)

		second, err := repo.List(ctx, ports.WorkspaceFilter{Limit: 1, Offset: 1})
		require.NoError(t, err)
		assert.Equal(t, 3, second.Total, "el total no depende de la página")
		require.Len(t, second.Workspaces, 1)
	})

	t.Run("filtra por período y por estado", func(t *testing.T) {
		byPeriod, err := repo.List(ctx, ports.WorkspaceFilter{PeriodID: period1.ID, Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, 2, byPeriod.Total)

		closed, err := repo.List(ctx, ports.WorkspaceFilter{Status: domain.WorkspaceClosed, Limit: 10})
		require.NoError(t, err)
		assert.Zero(t, closed.Total)
	})

	t.Run("un filtro con ID mal formado o desconocido no coincide con nada", func(t *testing.T) {
		for _, filter := range []ports.WorkspaceFilter{
			{PeriodID: "no-es-un-uuid", Limit: 10},
			{ProfessorID: "no-es-un-uuid", Limit: 10},
			{PeriodID: unknownID, Limit: 10},
		} {
			page, err := repo.List(ctx, filter)
			require.NoError(t, err)
			assert.Zero(t, page.Total)
			assert.Empty(t, page.Workspaces)
		}
	})

	t.Run("cierra una sola vez y conserva la primera fecha de cierre", func(t *testing.T) {
		closedAt := time.Date(2026, time.June, 12, 17, 0, 0, 0, time.UTC)
		require.NoError(t, repo.MarkClosed(ctx, bases.ID, closedAt))

		got, err := repo.GetByID(ctx, bases.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.WorkspaceClosed, got.Status)
		require.NotNil(t, got.ClosedAt)
		assert.True(t, got.ClosedAt.Equal(closedAt))
		assert.False(t, got.IsWritable())

		assert.ErrorIs(t, repo.MarkClosed(ctx, bases.ID, closedAt.Add(time.Hour)), domain.ErrWorkspaceClosed)
		again, err := repo.GetByID(ctx, bases.ID)
		require.NoError(t, err)
		assert.True(t, again.ClosedAt.Equal(closedAt), "el segundo intento no cambia la fecha de cierre")

		assert.ErrorIs(t, repo.MarkClosed(ctx, unknownID, closedAt), domain.ErrWorkspaceNotFound)
		assert.ErrorIs(t, repo.MarkClosed(ctx, "no-es-un-uuid", closedAt), domain.ErrWorkspaceNotFound)
	})

	t.Run("los cerrados siguen en el listado y se filtran por estado", func(t *testing.T) {
		closed, err := repo.List(ctx, ports.WorkspaceFilter{Status: domain.WorkspaceClosed, Limit: 10})
		require.NoError(t, err)
		require.Len(t, closed.Workspaces, 1)
		assert.Equal(t, bases.ID, closed.Workspaces[0].ID)

		all, err := repo.List(ctx, ports.WorkspaceFilter{ProfessorID: beto.ID, Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, 1, all.Total, "RF-03.6: un espacio cerrado se sigue consultando")
	})
}
