package workspaces_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dnunezd/SoftDevCloud/backend/internal/app/access"
	"github.com/dnunezd/SoftDevCloud/backend/internal/app/workspaces"
	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
	"github.com/dnunezd/SoftDevCloud/backend/internal/mocks"
	"github.com/dnunezd/SoftDevCloud/backend/internal/ports"
)

const (
	periodID    = "4a1b2c3d-5e6f-4a7b-8c9d-0e1f2a3b4c5d"
	workspaceID = "7e8f9a0b-1c2d-4e3f-8a4b-5c6d7e8f9a0b"
	profID      = "0b7c1c52-6a0e-4b63-8f0a-5a5f3b2d1c01"
	otherProfID = "2a3b4c5d-6e7f-4a8b-9c0d-1e2f3a4b5c6d"
	adminID     = "1d2e3f40-5a6b-4c7d-8e9f-a0b1c2d3e4f5"
)

var (
	// 10 de febrero de 2026, 07:00 en Bogotá: dentro de la ventana de creación.
	now = time.Date(2026, time.February, 10, 12, 0, 0, 0, time.UTC)

	professor = access.Principal{UserID: profID, Roles: []domain.Role{domain.RoleProfessor}}
	other     = access.Principal{UserID: otherProfID, Roles: []domain.Role{domain.RoleProfessor}}
	admin     = access.Principal{UserID: adminID, Roles: []domain.Role{domain.RoleAdmin}}
	monitor   = access.Principal{UserID: "m", Roles: []domain.Role{domain.RoleMonitor}}

	activePeriod = domain.AcademicPeriod{
		ID:               periodID,
		Code:             "2026-10",
		StartDate:        date(2026, time.February, 1),
		EndDate:          date(2026, time.June, 15),
		CreationDeadline: date(2026, time.February, 28),
		Status:           domain.PeriodActive,
	}
)

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func bogota(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("America/Bogota")
	require.NoError(t, err)
	return loc
}

type fixture struct {
	service    *workspaces.Service
	workspaces *mocks.WorkspaceRepository
	periods    *mocks.PeriodRepository
}

func newFixture(t *testing.T, at time.Time) fixture {
	f := fixture{workspaces: mocks.NewWorkspaceRepository(t), periods: mocks.NewPeriodRepository(t)}
	f.service = workspaces.NewService(f.workspaces, f.periods, fixedClock{at}, bogota(t))
	return f
}

func validInput() workspaces.CreateInput {
	return workspaces.CreateInput{
		Type: "curso", Name: "  Arquitectura de Software ", PeriodID: periodID,
		StartDate: "2026-02-02", EndDate: "2026-06-12", Observations: " Grupo nocturno ",
	}
}

func stored(status domain.WorkspaceStatus) domain.Workspace {
	w := domain.Workspace{
		ID: workspaceID, PeriodID: periodID, ProfessorID: profID, Type: domain.WorkspaceCourse,
		Name: "Arquitectura de Software", StartDate: date(2026, time.February, 2), EndDate: date(2026, time.June, 12),
		Status: status,
	}
	if status == domain.WorkspaceClosed {
		closedAt := now.Add(-time.Hour)
		w.ClosedAt = &closedAt
	}
	return w
}

// RF-03.1 y RF-03.3: el profesor crea el espacio y queda como responsable.
func TestCreate_RegistersTheWorkspaceForTheProfessor(t *testing.T) {
	f := newFixture(t, now)
	f.periods.EXPECT().GetByID(mock.Anything, periodID).Return(activePeriod, nil)
	f.workspaces.EXPECT().Create(mock.Anything, domain.Workspace{
		PeriodID: periodID, ProfessorID: profID, Type: domain.WorkspaceCourse, Name: "Arquitectura de Software",
		StartDate: date(2026, time.February, 2), EndDate: date(2026, time.June, 12),
		Observations: "Grupo nocturno", Status: domain.WorkspaceActive,
	}).RunAndReturn(func(_ context.Context, w domain.Workspace) (domain.Workspace, error) {
		w.ID = workspaceID
		return w, nil
	})

	created, err := f.service.Create(context.Background(), professor, validInput())

	require.NoError(t, err)
	assert.Equal(t, workspaceID, created.ID)
	assert.Equal(t, domain.WorkspaceActive, created.Status)
}

// RF-03.2 y RN-07: solo se crea dentro de la ventana de un período activo.
func TestCreate_EnforcesTheCreationWindow(t *testing.T) {
	closedPeriod := activePeriod
	closedPeriod.Status = domain.PeriodClosed

	tests := []struct {
		name    string
		at      time.Time
		period  domain.AcademicPeriod
		allowed bool
	}{
		{"dentro de la ventana", now, activePeriod, true},
		// 28 de febrero, 23:30 en Bogotá = 1 de marzo, 04:30 en UTC: la ventana sigue abierta.
		{"último día de la ventana según la zona académica", time.Date(2026, time.March, 1, 4, 30, 0, 0, time.UTC), activePeriod, true},
		{"un día después de la fecha límite", time.Date(2026, time.March, 1, 5, 0, 0, 0, time.UTC), activePeriod, false},
		{"período cerrado", now, closedPeriod, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t, tc.at)
			f.periods.EXPECT().GetByID(mock.Anything, periodID).Return(tc.period, nil)
			if tc.allowed {
				f.workspaces.EXPECT().Create(mock.Anything, mock.Anything).Return(stored(domain.WorkspaceActive), nil)
			}

			_, err := f.service.Create(context.Background(), professor, validInput())

			if tc.allowed {
				require.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, domain.ErrCreationWindowClosed)
		})
	}
}

func TestCreate_RejectsInvalidData(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*workspaces.CreateInput)
		fields []string
	}{
		{"tipo desconocido", func(in *workspaces.CreateInput) { in.Type = "semillero" }, []string{"type"}},
		{"nombre vacío", func(in *workspaces.CreateInput) { in.Name = " " }, []string{"name"}},
		{"sin período", func(in *workspaces.CreateInput) { in.PeriodID = "" }, []string{"period_id"}},
		{"fecha con formato inválido", func(in *workspaces.CreateInput) { in.StartDate = "febrero" }, []string{"start_date"}},
		{"cierre antes del inicio", func(in *workspaces.CreateInput) { in.EndDate = "2026-01-01" }, []string{"end_date"}},
		{"varios problemas a la vez", func(in *workspaces.CreateInput) {
			in.Type, in.Name, in.EndDate = "", "", "x"
		}, []string{"type", "name", "end_date"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t, now) // sin expectativas: nada se consulta ni se guarda
			in := validInput()
			tc.mutate(&in)

			_, err := f.service.Create(context.Background(), professor, in)

			var derr *domain.Error
			require.ErrorAs(t, err, &derr)
			assert.Equal(t, domain.KindValidation, derr.Kind)
			for _, field := range tc.fields {
				assert.Contains(t, derr.Fields, field)
			}
		})
	}
}

func TestCreate_PeriodMustExist(t *testing.T) {
	f := newFixture(t, now)
	f.periods.EXPECT().GetByID(mock.Anything, periodID).Return(domain.AcademicPeriod{}, domain.ErrPeriodNotFound)

	_, err := f.service.Create(context.Background(), professor, validInput())

	var derr *domain.Error
	require.ErrorAs(t, err, &derr)
	assert.Equal(t, domain.KindValidation, derr.Kind)
	assert.Contains(t, derr.Fields, "period_id")
}

func TestCreate_OnlyProfessorsCreate(t *testing.T) {
	for name, principal := range map[string]access.Principal{"administrador": admin, "monitor": monitor} {
		t.Run(name, func(t *testing.T) {
			_, err := newFixture(t, now).service.Create(context.Background(), principal, validInput())

			assert.ErrorIs(t, err, domain.ErrForbidden)
		})
	}
}

// RF-03.4, RF-12.1 y RN-05: el profesor lista solo lo suyo; el administrador, todo.
func TestList_ScopesByRole(t *testing.T) {
	t.Run("el profesor solo ve sus espacios", func(t *testing.T) {
		f := newFixture(t, now)
		f.workspaces.EXPECT().List(mock.Anything, ports.WorkspaceFilter{ProfessorID: profID, Limit: workspaces.DefaultPageSize}).
			Return(ports.WorkspacePage{Workspaces: []domain.Workspace{stored(domain.WorkspaceActive)}, Total: 1}, nil)

		page, err := f.service.List(context.Background(), professor, workspaces.ListInput{})

		require.NoError(t, err)
		assert.Equal(t, 1, page.Total)
		assert.Equal(t, workspaces.DefaultPageSize, page.Limit)
	})

	t.Run("el administrador ve todos y puede filtrar", func(t *testing.T) {
		f := newFixture(t, now)
		f.workspaces.EXPECT().List(mock.Anything, ports.WorkspaceFilter{
			PeriodID: periodID, Status: domain.WorkspaceClosed, Limit: 10, Offset: 20,
		}).Return(ports.WorkspacePage{}, nil)

		_, err := f.service.List(context.Background(), admin, workspaces.ListInput{
			PeriodID: periodID, Status: "cerrado", Limit: 10, Offset: 20,
		})

		require.NoError(t, err)
	})

	t.Run("quien es profesor y administrador a la vez ve todos", func(t *testing.T) {
		f := newFixture(t, now)
		both := access.Principal{UserID: profID, Roles: []domain.Role{domain.RoleProfessor, domain.RoleAdmin}}
		f.workspaces.EXPECT().List(mock.Anything, ports.WorkspaceFilter{Limit: workspaces.DefaultPageSize}).Return(ports.WorkspacePage{}, nil)

		_, err := f.service.List(context.Background(), both, workspaces.ListInput{})

		require.NoError(t, err)
	})

	t.Run("otros roles no listan espacios", func(t *testing.T) {
		_, err := newFixture(t, now).service.List(context.Background(), monitor, workspaces.ListInput{})

		assert.ErrorIs(t, err, domain.ErrForbidden)
	})
}

func TestList_RejectsInvalidCriteria(t *testing.T) {
	_, err := newFixture(t, now).service.List(context.Background(), professor, workspaces.ListInput{Status: "VIEJO", Limit: 999, Offset: -1})

	var derr *domain.Error
	require.ErrorAs(t, err, &derr)
	assert.Equal(t, domain.KindValidation, derr.Kind)
	assert.Len(t, derr.Fields, 3)
}

// RF-12.1: un profesor no puede ni confirmar que existe un espacio ajeno.
func TestGet_Visibility(t *testing.T) {
	tests := []struct {
		name      string
		principal access.Principal
		err       error
	}{
		{"su profesor responsable", professor, nil},
		{"el administrador", admin, nil},
		{"otro profesor", other, domain.ErrWorkspaceNotFound},
		{"un monitor", monitor, domain.ErrWorkspaceNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t, now)
			f.workspaces.EXPECT().GetByID(mock.Anything, workspaceID).Return(stored(domain.WorkspaceActive), nil)

			workspace, err := f.service.Get(context.Background(), tc.principal, workspaceID)

			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, workspaceID, workspace.ID)
		})
	}
}

// RF-03.6: un espacio cerrado se sigue consultando.
func TestGet_ClosedWorkspaceIsReadable(t *testing.T) {
	f := newFixture(t, now)
	f.workspaces.EXPECT().GetByID(mock.Anything, workspaceID).Return(stored(domain.WorkspaceClosed), nil)

	workspace, err := f.service.Get(context.Background(), professor, workspaceID)

	require.NoError(t, err)
	assert.Equal(t, domain.WorkspaceClosed, workspace.Status)
	assert.NotNil(t, workspace.ClosedAt)
}

// RF-03.5 y RN-09: el cierre lo hace el profesor responsable y no se repite.
func TestClose(t *testing.T) {
	t.Run("el profesor responsable cierra su espacio", func(t *testing.T) {
		f := newFixture(t, now)
		f.workspaces.EXPECT().GetByID(mock.Anything, workspaceID).Return(stored(domain.WorkspaceActive), nil)
		f.workspaces.EXPECT().MarkClosed(mock.Anything, workspaceID, now).Return(nil)

		closed, err := f.service.Close(context.Background(), professor, workspaceID)

		require.NoError(t, err)
		assert.Equal(t, domain.WorkspaceClosed, closed.Status)
		require.NotNil(t, closed.ClosedAt)
		assert.Equal(t, now, *closed.ClosedAt)
		assert.False(t, closed.IsWritable())
	})

	t.Run("otro profesor no puede ni ver que existe", func(t *testing.T) {
		f := newFixture(t, now)
		f.workspaces.EXPECT().GetByID(mock.Anything, workspaceID).Return(stored(domain.WorkspaceActive), nil)

		_, err := f.service.Close(context.Background(), other, workspaceID)

		assert.ErrorIs(t, err, domain.ErrWorkspaceNotFound)
	})

	t.Run("el administrador ve el espacio pero no lo cierra", func(t *testing.T) {
		f := newFixture(t, now)
		f.workspaces.EXPECT().GetByID(mock.Anything, workspaceID).Return(stored(domain.WorkspaceActive), nil)

		_, err := f.service.Close(context.Background(), admin, workspaceID)

		assert.ErrorIs(t, err, domain.ErrForbidden)
	})

	t.Run("un espacio cerrado no se cierra otra vez", func(t *testing.T) {
		f := newFixture(t, now)
		f.workspaces.EXPECT().GetByID(mock.Anything, workspaceID).Return(stored(domain.WorkspaceClosed), nil)

		_, err := f.service.Close(context.Background(), professor, workspaceID)

		assert.ErrorIs(t, err, domain.ErrWorkspaceClosed)
	})

	t.Run("otro cierre se adelantó entre la lectura y la escritura", func(t *testing.T) {
		f := newFixture(t, now)
		f.workspaces.EXPECT().GetByID(mock.Anything, workspaceID).Return(stored(domain.WorkspaceActive), nil)
		f.workspaces.EXPECT().MarkClosed(mock.Anything, workspaceID, now).Return(domain.ErrWorkspaceClosed)

		_, err := f.service.Close(context.Background(), professor, workspaceID)

		assert.ErrorIs(t, err, domain.ErrWorkspaceClosed)
	})
}
