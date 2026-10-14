package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
)

func intPtr(v int) *int { return &v }

func rolePtr(r domain.Role) *domain.Role { return &r }

func TestAssignment_Validate(t *testing.T) {
	tests := []struct {
		name       string
		assignment domain.Assignment
		field      string
	}{
		{"AG de 22 h es válida", domain.Assignment{Role: domain.RoleGraduateAssistant, WeeklyHours: 22}, ""},
		{"monitor de 12 h es válido", domain.Assignment{Role: domain.RoleMonitor, WeeklyHours: 12}, ""},
		{"AG de 23 h", domain.Assignment{Role: domain.RoleGraduateAssistant, WeeklyHours: 23}, "weekly_hours"},
		{"monitor de 13 h", domain.Assignment{Role: domain.RoleMonitor, WeeklyHours: 13}, "weekly_hours"},
		{"cero horas", domain.Assignment{Role: domain.RoleMonitor, WeeklyHours: 0}, "weekly_hours"},
		{"horas negativas", domain.Assignment{Role: domain.RoleMonitor, WeeklyHours: -2}, "weekly_hours"},
		{"rol global que no es de vinculación", domain.Assignment{Role: domain.RoleProfessor, WeeklyHours: 4}, "role"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.assignment.Validate()

			if tc.field == "" {
				assert.NoError(t, err)
				return
			}
			var derr *domain.Error
			require.ErrorAs(t, err, &derr)
			assert.Contains(t, derr.Fields, tc.field)
		})
	}
}

func TestAssignment_IsActiveFollowsWorkspace(t *testing.T) {
	assignment := domain.Assignment{WorkspaceID: "ws-1"}

	assert.True(t, assignment.IsActive(activeWorkspace))
	assert.False(t, assignment.IsActive(closedWorkspace))
}

func TestAssignment_Rectify(t *testing.T) {
	now := at(2026, time.October, 14, 12, 0)
	original := domain.Assignment{ID: "asg-1", UserID: "u-1", WorkspaceID: "ws-1", Role: domain.RoleMonitor, WeeklyHours: 4}

	t.Run("corrige las horas y registra el antes y el después", func(t *testing.T) {
		updated, audit, err := original.Rectify(domain.AssignmentChanges{WeeklyHours: intPtr(6)}, "admin-1", now)

		require.NoError(t, err)
		assert.Equal(t, 6, updated.WeeklyHours)
		assert.Equal(t, domain.RoleMonitor, updated.Role)
		assert.Equal(t, 4, original.WeeklyHours, "no modifica el original")
		assert.Equal(t, domain.AssignmentAudit{
			AssignmentID: "asg-1",
			AdminID:      "admin-1",
			Before:       original,
			After:        updated,
			ChangedAt:    now,
		}, audit)
	})
	t.Run("puede corregir el rol", func(t *testing.T) {
		updated, _, err := original.Rectify(domain.AssignmentChanges{Role: rolePtr(domain.RoleGraduateAssistant)}, "admin-1", now)

		require.NoError(t, err)
		assert.Equal(t, domain.RoleGraduateAssistant, updated.Role)
	})
	t.Run("exige indicar algún cambio", func(t *testing.T) {
		_, _, err := original.Rectify(domain.AssignmentChanges{}, "admin-1", now)

		var derr *domain.Error
		require.ErrorAs(t, err, &derr)
		assert.Contains(t, derr.Fields, "changes")
	})
	t.Run("rechaza un resultado inválido", func(t *testing.T) {
		_, _, err := original.Rectify(domain.AssignmentChanges{WeeklyHours: intPtr(13)}, "admin-1", now)

		var derr *domain.Error
		require.ErrorAs(t, err, &derr)
		assert.Contains(t, derr.Fields, "weekly_hours")
	})
}

func TestWeeklyReport_Transitions(t *testing.T) {
	now := at(2026, time.October, 20, 8, 0)

	t.Run("flujo exitoso", func(t *testing.T) {
		report := domain.WeeklyReport{Status: domain.ReportPending}

		require.NoError(t, report.MarkProcessing())
		require.NoError(t, report.MarkCompleted("reports/2026-10-12/u-1.pdf", now))

		assert.Equal(t, domain.ReportCompleted, report.Status)
		assert.Equal(t, "reports/2026-10-12/u-1.pdf", report.StorageKey)
		require.NotNil(t, report.GeneratedAt)
		assert.Equal(t, now, *report.GeneratedAt)
	})
	t.Run("un reintento mantiene PROCESANDO", func(t *testing.T) {
		report := domain.WeeklyReport{Status: domain.ReportProcessing}

		assert.NoError(t, report.MarkProcessing())
		assert.Equal(t, domain.ReportProcessing, report.Status)
	})
	t.Run("falla tras agotar reintentos", func(t *testing.T) {
		report := domain.WeeklyReport{Status: domain.ReportProcessing}

		require.NoError(t, report.MarkFailed("la IA no respondió"))

		assert.Equal(t, domain.ReportFailed, report.Status)
		assert.Equal(t, "la IA no respondió", report.Error)
	})
	t.Run("completar exige PDF almacenado", func(t *testing.T) {
		report := domain.WeeklyReport{Status: domain.ReportProcessing}

		err := report.MarkCompleted("  ", now)

		var derr *domain.Error
		require.ErrorAs(t, err, &derr)
		assert.Contains(t, derr.Fields, "storage_key")
		assert.Equal(t, domain.ReportProcessing, report.Status)
		assert.Nil(t, report.GeneratedAt)
	})
	t.Run("transiciones inválidas", func(t *testing.T) {
		completed := domain.WeeklyReport{Status: domain.ReportCompleted, StorageKey: "k"}
		failed := domain.WeeklyReport{Status: domain.ReportFailed}
		pending := domain.WeeklyReport{Status: domain.ReportPending}

		assert.ErrorIs(t, completed.MarkProcessing(), domain.ErrInvalidReportState, "COMPLETADO es final")
		assert.ErrorIs(t, completed.MarkFailed("x"), domain.ErrInvalidReportState)
		assert.ErrorIs(t, failed.MarkProcessing(), domain.ErrInvalidReportState, "FALLIDO es final")
		assert.ErrorIs(t, pending.MarkCompleted("k", now), domain.ErrInvalidReportState, "no se completa sin pasar por PROCESANDO")
	})
}

func TestReportBatch_RefreshStatus(t *testing.T) {
	tests := []struct {
		name    string
		reports []domain.ReportStatus
		want    domain.BatchStatus
	}{
		{"sin reportes aún", nil, domain.BatchInProgress},
		{"todos pendientes", []domain.ReportStatus{domain.ReportPending, domain.ReportPending}, domain.BatchInProgress},
		{"alguno procesando", []domain.ReportStatus{domain.ReportCompleted, domain.ReportProcessing}, domain.BatchInProgress},
		{"uno falló pero otro sigue pendiente", []domain.ReportStatus{domain.ReportFailed, domain.ReportPending}, domain.BatchInProgress},
		{"todos completados", []domain.ReportStatus{domain.ReportCompleted, domain.ReportCompleted}, domain.BatchCompleted},
		{"terminaron y alguno falló", []domain.ReportStatus{domain.ReportCompleted, domain.ReportFailed}, domain.BatchCompletedWithErrors},
		{"todos fallaron", []domain.ReportStatus{domain.ReportFailed, domain.ReportFailed}, domain.BatchCompletedWithErrors},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reports := make([]domain.WeeklyReport, len(tc.reports))
			for i, status := range tc.reports {
				reports[i] = domain.WeeklyReport{Status: status}
			}
			batch := domain.ReportBatch{Status: domain.BatchCompleted}

			batch.RefreshStatus(reports)

			assert.Equal(t, tc.want, batch.Status)
		})
	}
}
