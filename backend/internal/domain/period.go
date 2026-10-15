package domain

import (
	"strings"
	"time"
)

// PeriodStatus es el estado de un período académico.
type PeriodStatus string

const (
	PeriodActive PeriodStatus = "ACTIVO"
	PeriodClosed PeriodStatus = "CERRADO"
)

// AcademicPeriod es un período académico (p. ej. "2026-10", "2026-20") con una
// ventana para crear espacios (RF-03.2, RN-07).
//
// Las fechas son de calendario: la hora y la zona se ignoran al compararlas.
type AcademicPeriod struct {
	ID               string
	Code             string
	StartDate        time.Time
	EndDate          time.Time
	CreationDeadline time.Time
	Status           PeriodStatus
}

// Validate comprueba la coherencia del período: código, estado y
// inicio <= fecha límite de creación <= cierre.
func (p AcademicPeriod) Validate() error {
	fields := map[string]string{}
	if strings.TrimSpace(p.Code) == "" {
		fields["code"] = "es obligatorio"
	}
	if p.Status != PeriodActive && p.Status != PeriodClosed {
		fields["status"] = "debe ser ACTIVO o CERRADO"
	}
	start, deadline, end := calendarDay(p.StartDate), calendarDay(p.CreationDeadline), calendarDay(p.EndDate)
	if deadline.Before(start) || deadline.After(end) {
		fields["creation_deadline"] = "debe estar entre la fecha de inicio y la de cierre del período"
	}
	if end.Before(start) {
		fields["end_date"] = "no puede ser anterior a la fecha de inicio"
	}
	if len(fields) > 0 {
		return NewValidationError(fields)
	}
	return nil
}

// Close cierra el período: deja de aceptar la creación de espacios (RF-03.2).
// Cerrar un período ya cerrado es un error: el cierre no se repite.
func (p *AcademicPeriod) Close() error {
	if p.Status == PeriodClosed {
		return ErrPeriodClosed
	}
	p.Status = PeriodClosed
	return nil
}

// AcceptsWorkspaceCreation indica si hoy se pueden crear cursos o proyectos en
// el período: debe estar ACTIVO y no haber pasado su fecha límite de creación
// (RF-03.2, RN-07). today se interpreta como fecha de calendario.
func (p AcademicPeriod) AcceptsWorkspaceCreation(today time.Time) bool {
	return p.Status == PeriodActive && !calendarDay(today).After(calendarDay(p.CreationDeadline))
}
