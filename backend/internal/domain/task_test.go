package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
)

var (
	activeWorkspace = domain.Workspace{ID: "ws-1", Status: domain.WorkspaceActive}
	closedWorkspace = domain.Workspace{ID: "ws-1", Status: domain.WorkspaceClosed}

	validTask = domain.TaskInput{
		Title:        "  Revisión de talleres ",
		Description:  "Calificación del taller 3",
		Status:       "abierto",
		HoursSpent:   4,
		Observations: "Pendiente entregar notas",
	}
)

// currentWeek es la semana del 12 al 18 de octubre de 2026.
func currentWeek() domain.Week {
	return domain.WeekContaining(at(2026, time.October, 14, 9, 0), bogota)
}

func TestNewTask_OnTimeInActiveWeek(t *testing.T) {
	now := at(2026, time.October, 14, 9, 0)

	task, err := domain.NewTask("asg-1", currentWeek(), validTask, activeWorkspace, now)

	require.NoError(t, err)
	assert.Equal(t, "asg-1", task.AssignmentID)
	assert.Equal(t, "Revisión de talleres", task.Title)
	assert.Equal(t, domain.TaskOpen, task.Status)
	assert.Equal(t, 4, task.HoursSpent)
	assert.False(t, task.IsLate)
	assert.Equal(t, now, task.CreatedAt)
	assert.Equal(t, now, task.UpdatedAt)
}

func TestNewTask_IsLateWhenWeekAlreadyClosed(t *testing.T) {
	// Lunes 19 a las 00:00: la semana del 12 ya cerró.
	now := at(2026, time.October, 19, 0, 0)

	task, err := domain.NewTask("asg-1", currentWeek(), validTask, activeWorkspace, now)

	require.NoError(t, err)
	assert.True(t, task.IsLate, "RN-32: toda tarea de una semana cerrada es reporte tardío")
}

func TestNewTask_Rejections(t *testing.T) {
	now := at(2026, time.October, 14, 9, 0)
	nextWeek := domain.WeekContaining(at(2026, time.October, 21, 9, 0), bogota)

	t.Run("espacio cerrado", func(t *testing.T) {
		_, err := domain.NewTask("asg-1", currentWeek(), validTask, closedWorkspace, now)
		assert.ErrorIs(t, err, domain.ErrWorkspaceClosed)
	})
	t.Run("semana futura", func(t *testing.T) {
		_, err := domain.NewTask("asg-1", nextWeek, validTask, activeWorkspace, now)

		var derr *domain.Error
		require.ErrorAs(t, err, &derr)
		assert.Equal(t, map[string]string{"week_start": "no puede ser una semana futura"}, derr.Fields)
	})
	t.Run("datos inválidos", func(t *testing.T) {
		_, err := domain.NewTask("asg-1", currentWeek(), domain.TaskInput{Title: " ", Status: "CANCELADO", HoursSpent: -1}, activeWorkspace, now)

		var derr *domain.Error
		require.ErrorAs(t, err, &derr)
		assert.Equal(t, domain.KindValidation, derr.Kind)
		assert.Contains(t, derr.Fields, "title")
		assert.Contains(t, derr.Fields, "status")
		assert.Contains(t, derr.Fields, "hours_spent")
	})
}

func TestNewTask_AllowsZeroHours(t *testing.T) {
	in := validTask
	in.HoursSpent = 0

	_, err := domain.NewTask("asg-1", currentWeek(), in, activeWorkspace, at(2026, time.October, 14, 9, 0))

	assert.NoError(t, err, "el CHECK de la BD admite hours_spent >= 0")
}

func TestTask_CanModifyOnlyDuringActiveWeek(t *testing.T) {
	created := at(2026, time.October, 14, 9, 0)
	task, err := domain.NewTask("asg-1", currentWeek(), validTask, activeWorkspace, created)
	require.NoError(t, err)

	tests := []struct {
		name      string
		now       time.Time
		workspace domain.Workspace
		want      bool
	}{
		{"semana activa", at(2026, time.October, 16, 10, 0), activeWorkspace, true},
		{"domingo 23:59 sigue activa", at(2026, time.October, 18, 23, 59), activeWorkspace, true},
		{"lunes siguiente 00:00: semana cerrada", at(2026, time.October, 19, 0, 0), activeWorkspace, false},
		{"semana activa pero espacio cerrado", at(2026, time.October, 16, 10, 0), closedWorkspace, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, task.CanModify(tc.now, tc.workspace))
		})
	}
}

func TestTask_LateReportIsNeverModifiable(t *testing.T) {
	// Creada el lunes 19 para la semana del 12: tardía. Aun dentro de su propia semana "activa" lógica
	// no existe forma de modificarla, porque la semana de la tarea ya cerró.
	now := at(2026, time.October, 20, 9, 0)
	task, err := domain.NewTask("asg-1", currentWeek(), validTask, activeWorkspace, now)
	require.NoError(t, err)
	require.True(t, task.IsLate)

	assert.False(t, task.CanModify(now, activeWorkspace))
	_, err = task.Update(validTask, now, activeWorkspace)
	assert.ErrorIs(t, err, domain.ErrTaskNotModifiable)
}

func TestTask_Update(t *testing.T) {
	created := at(2026, time.October, 14, 9, 0)
	task, err := domain.NewTask("asg-1", currentWeek(), validTask, activeWorkspace, created)
	require.NoError(t, err)
	later := at(2026, time.October, 16, 17, 0)

	t.Run("actualiza datos y conserva semana y marca de tardío", func(t *testing.T) {
		updated, err := task.Update(domain.TaskInput{Title: "Taller 3 calificado", Status: "FINALIZADO", HoursSpent: 6}, later, activeWorkspace)

		require.NoError(t, err)
		assert.Equal(t, "Taller 3 calificado", updated.Title)
		assert.Equal(t, domain.TaskDone, updated.Status)
		assert.Equal(t, 6, updated.HoursSpent)
		assert.Equal(t, later, updated.UpdatedAt)
		assert.True(t, updated.WeekStart.Equal(task.WeekStart), "RN-26: la semana nunca cambia")
		assert.Equal(t, task.CreatedAt, updated.CreatedAt)
		assert.False(t, updated.IsLate)
		assert.Equal(t, "Revisión de talleres", task.Title, "no modifica el original")
	})
	t.Run("rechaza fuera de la semana activa", func(t *testing.T) {
		_, err := task.Update(validTask, at(2026, time.October, 20, 9, 0), activeWorkspace)
		assert.ErrorIs(t, err, domain.ErrTaskNotModifiable)
	})
	t.Run("rechaza con el espacio cerrado", func(t *testing.T) {
		_, err := task.Update(validTask, later, closedWorkspace)
		assert.ErrorIs(t, err, domain.ErrWorkspaceClosed)
	})
	t.Run("rechaza datos inválidos", func(t *testing.T) {
		_, err := task.Update(domain.TaskInput{Title: "", Status: "ABIERTO"}, later, activeWorkspace)

		var derr *domain.Error
		require.ErrorAs(t, err, &derr)
		assert.Contains(t, derr.Fields, "title")
	})
}

func TestTaskStatus_Valid(t *testing.T) {
	for _, status := range []domain.TaskStatus{domain.TaskOpen, domain.TaskInProgress, domain.TaskDone} {
		assert.True(t, status.Valid(), status)
	}
	for _, status := range []domain.TaskStatus{"", "CERRADO", "abierto"} {
		assert.False(t, status.Valid(), status)
	}
}
