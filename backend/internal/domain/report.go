package domain

import (
	"strings"
	"time"
)

// ReportStatus es el estado de un reporte semanal individual.
type ReportStatus string

const (
	ReportPending    ReportStatus = "PENDIENTE"
	ReportProcessing ReportStatus = "PROCESANDO"
	ReportCompleted  ReportStatus = "COMPLETADO"
	ReportFailed     ReportStatus = "FALLIDO"
)

// BatchStatus es el estado de un lote de reportes. Se almacena, pero siempre se
// recalcula a partir del estado de los reportes del lote (RefreshStatus).
type BatchStatus string

const (
	BatchInProgress          BatchStatus = "EN_PROCESO"
	BatchCompleted           BatchStatus = "COMPLETADO"
	BatchCompletedWithErrors BatchStatus = "COMPLETADO_CON_ERRORES"
)

// WeeklyReport es el reporte PDF de una persona para una semana, generado con
// IA a partir solo de tareas, descripciones, observaciones y horas (RN-43). Su
// propósito es sintetizar el trabajo, no evaluar el desempeño (RN-46).
//
// Transiciones: PENDIENTE -> PROCESANDO -> COMPLETADO | FALLIDO. Un reintento
// del trabajo mantiene el reporte en PROCESANDO. COMPLETADO y FALLIDO son
// finales, y COMPLETADO implica un PDF almacenado (StorageKey).
type WeeklyReport struct {
	ID          string
	BatchID     string
	UserID      string
	WeekStart   Week
	Status      ReportStatus
	StorageKey  string
	Error       string
	GeneratedAt *time.Time
}

// MarkProcessing registra que el worker empezó a generar el reporte.
func (r *WeeklyReport) MarkProcessing() error {
	if r.Status != ReportPending && r.Status != ReportProcessing {
		return ErrInvalidReportState
	}
	r.Status = ReportProcessing
	return nil
}

// MarkCompleted registra que el PDF quedó almacenado en storageKey.
func (r *WeeklyReport) MarkCompleted(storageKey string, now time.Time) error {
	if r.Status != ReportProcessing {
		return ErrInvalidReportState
	}
	if strings.TrimSpace(storageKey) == "" {
		return NewValidationError(map[string]string{"storage_key": "es obligatoria para completar el reporte"})
	}
	r.Status = ReportCompleted
	r.StorageKey = storageKey
	r.Error = ""
	r.GeneratedAt = &now
	return nil
}

// MarkFailed registra que la generación falló de forma definitiva, tras agotar
// los reintentos.
func (r *WeeklyReport) MarkFailed(message string) error {
	if r.Status != ReportPending && r.Status != ReportProcessing {
		return ErrInvalidReportState
	}
	r.Status = ReportFailed
	r.Error = message
	return nil
}

// ReportBatch es la solicitud de un profesor para generar los reportes de una
// semana: produce un reporte por persona vinculada (RN-41, RN-42, RN-44).
type ReportBatch struct {
	ID          string
	ProfessorID string
	WeekStart   Week
	Status      BatchStatus
	RequestedAt time.Time
}

// RefreshStatus recalcula el estado del lote a partir de sus reportes:
// EN_PROCESO mientras alguno esté pendiente o en proceso (o aún no existan),
// COMPLETADO_CON_ERRORES si ya terminaron y alguno falló, y COMPLETADO si todos
// terminaron bien.
func (b *ReportBatch) RefreshStatus(reports []WeeklyReport) {
	if len(reports) == 0 {
		b.Status = BatchInProgress
		return
	}
	failed := false
	for _, report := range reports {
		switch report.Status {
		case ReportCompleted:
		case ReportFailed:
			failed = true
		default:
			b.Status = BatchInProgress
			return
		}
	}
	if failed {
		b.Status = BatchCompletedWithErrors
		return
	}
	b.Status = BatchCompleted
}
