package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestAcademicPeriod_AcceptsWorkspaceCreation(t *testing.T) {
	period := domain.AcademicPeriod{
		Code:             "2026-20",
		StartDate:        date(2026, time.August, 1),
		EndDate:          date(2026, time.December, 15),
		CreationDeadline: date(2026, time.August, 31),
		Status:           domain.PeriodActive,
	}
	closed := period
	closed.Status = domain.PeriodClosed

	tests := []struct {
		name   string
		period domain.AcademicPeriod
		today  time.Time
		want   bool
	}{
		{"dentro de la ventana", period, at(2026, time.August, 15, 10, 0), true},
		{"último día de la ventana, a cualquier hora", period, at(2026, time.August, 31, 23, 59), true},
		{"un día después de la fecha límite", period, at(2026, time.September, 1, 0, 0), false},
		{"período cerrado", closed, at(2026, time.August, 15, 10, 0), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.period.AcceptsWorkspaceCreation(tc.today))
		})
	}
}

func TestAcademicPeriod_Validate(t *testing.T) {
	valid := domain.AcademicPeriod{
		Code:             "2026-10",
		StartDate:        date(2026, time.February, 1),
		EndDate:          date(2026, time.June, 15),
		CreationDeadline: date(2026, time.February, 28),
		Status:           domain.PeriodActive,
	}
	require.NoError(t, valid.Validate())

	tests := []struct {
		name   string
		mutate func(*domain.AcademicPeriod)
		field  string
	}{
		{"código vacío", func(p *domain.AcademicPeriod) { p.Code = " " }, "code"},
		{"estado desconocido", func(p *domain.AcademicPeriod) { p.Status = "INACTIVO" }, "status"},
		{"fecha límite antes del inicio", func(p *domain.AcademicPeriod) { p.CreationDeadline = date(2026, time.January, 31) }, "creation_deadline"},
		{"fecha límite después del cierre", func(p *domain.AcademicPeriod) { p.CreationDeadline = date(2026, time.June, 16) }, "creation_deadline"},
		{"cierre antes del inicio", func(p *domain.AcademicPeriod) { p.EndDate = date(2026, time.January, 1) }, "end_date"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			period := valid
			tc.mutate(&period)

			var derr *domain.Error
			require.ErrorAs(t, period.Validate(), &derr)
			assert.Contains(t, derr.Fields, tc.field)
		})
	}
}

func TestWorkspace_Validate(t *testing.T) {
	valid := domain.Workspace{
		Type:      domain.WorkspaceCourse,
		Name:      "Arquitectura de Software",
		StartDate: date(2026, time.August, 3),
		EndDate:   date(2026, time.December, 4),
		Status:    domain.WorkspaceActive,
	}
	require.NoError(t, valid.Validate())

	project := valid
	project.Type = domain.WorkspaceProject
	require.NoError(t, project.Validate())

	tests := []struct {
		name   string
		mutate func(*domain.Workspace)
		field  string
	}{
		{"tipo desconocido", func(w *domain.Workspace) { w.Type = "SEMILLERO" }, "type"},
		{"nombre vacío", func(w *domain.Workspace) { w.Name = "  " }, "name"},
		{"cierre antes del inicio", func(w *domain.Workspace) { w.EndDate = date(2026, time.July, 1) }, "end_date"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			workspace := valid
			tc.mutate(&workspace)

			var derr *domain.Error
			require.ErrorAs(t, workspace.Validate(), &derr)
			assert.Contains(t, derr.Fields, tc.field)
		})
	}
}

func TestAcademicPeriod_Close(t *testing.T) {
	period := domain.AcademicPeriod{
		Code:             "2026-10",
		StartDate:        date(2026, time.February, 1),
		EndDate:          date(2026, time.June, 15),
		CreationDeadline: date(2026, time.February, 28),
		Status:           domain.PeriodActive,
	}
	require.True(t, period.AcceptsWorkspaceCreation(at(2026, time.February, 10, 8, 0)))

	require.NoError(t, period.Close())

	assert.Equal(t, domain.PeriodClosed, period.Status)
	assert.False(t, period.AcceptsWorkspaceCreation(at(2026, time.February, 10, 8, 0)), "RF-03.2: un período cerrado no admite crear espacios")
	assert.ErrorIs(t, period.Close(), domain.ErrPeriodClosed, "no se cierra dos veces")
}

func TestWorkspace_Close(t *testing.T) {
	now := at(2026, time.December, 5, 8, 0)
	workspace := domain.Workspace{ID: "ws-1", Status: domain.WorkspaceActive}
	require.True(t, workspace.IsWritable())

	require.NoError(t, workspace.Close(now))

	assert.Equal(t, domain.WorkspaceClosed, workspace.Status)
	require.NotNil(t, workspace.ClosedAt)
	assert.Equal(t, now, *workspace.ClosedAt)
	assert.False(t, workspace.IsWritable(), "RN-09: un espacio cerrado es de solo lectura")
	assert.ErrorIs(t, workspace.Close(now.Add(time.Hour)), domain.ErrWorkspaceClosed, "no se cierra dos veces")
	assert.Equal(t, now, *workspace.ClosedAt, "el segundo intento no cambia la fecha de cierre")
}
